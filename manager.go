package sandwich

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/Sandwich-Daemon/pkg/syncmap"
	"github.com/coder/websocket"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

type Manager struct {
	logger *slog.Logger

	sandwich      *Sandwich
	configuration *atomic.Pointer[ManagerConfiguration]

	gateway                           *atomic.Pointer[discord.GatewayBotResponse]
	gatewaySessionStartLimitRemaining *atomic.Int32

	user *atomic.Pointer[discord.User]

	producer Producer

	shardCount *atomic.Int32

	ready   chan struct{}
	readyWg sync.WaitGroup

	shards *syncmap.Map[int32, *Shard]
	guilds *csmap.CsMap[discord.Snowflake, bool]

	startedAt *atomic.Pointer[time.Time]
}

func NewManager(s *Sandwich, config *ManagerConfiguration) *Manager {
	manager := &Manager{
		logger: s.logger.With("manager", config.ApplicationIdentifier),

		sandwich:      s,
		configuration: &atomic.Pointer[ManagerConfiguration]{},

		gateway:                           &atomic.Pointer[discord.GatewayBotResponse]{},
		gatewaySessionStartLimitRemaining: &atomic.Int32{},

		user: &atomic.Pointer[discord.User]{},

		producer: nil,

		shardCount: &atomic.Int32{},

		ready:   make(chan struct{}),
		readyWg: sync.WaitGroup{},

		shards: &syncmap.Map[int32, *Shard]{},
		guilds: csmap.Create[discord.Snowflake, bool](),

		startedAt: &atomic.Pointer[time.Time]{},
	}

	manager.configuration.Store(config)

	return manager
}

// Initialize initializes the manager. This includes checking the gateway
func (m *Manager) Initialize(ctx context.Context) error {
	m.sandwich.gatewayLimiter.Lock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discord.EndpointGateway, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.sandwich.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}

	defer resp.Body.Close()

	var gatewayBotResponse discord.GatewayBotResponse
	if err := json.NewDecoder(resp.Body).Decode(&gatewayBotResponse); err != nil {
		return fmt.Errorf("failed to decode gateway bot response: %w", err)
	}

	m.gateway.Store(&gatewayBotResponse)
	m.gatewaySessionStartLimitRemaining.Store(gatewayBotResponse.SessionStartLimit.Remaining)

	configuration := m.configuration.Load()

	clientName := configuration.ClientName

	// If the client name includes a random suffix, we need to add a random suffix to the client name.
	if configuration.IncludeRandomSuffix {
		clientName = fmt.Sprintf("%s-%s", clientName, randomHex(8))
	}

	producer, err := m.sandwich.producerProvider.GetProducer(ctx, configuration.ApplicationIdentifier, clientName)
	if err != nil {
		return fmt.Errorf("failed to get producer: %w", err)
	}

	m.producer = producer

	return nil
}

func (m *Manager) Start(ctx context.Context) error {
	configuration := m.configuration.Load()

	shardIDs, shardCount := m.getInitialShardCount(
		configuration.ShardCount,
		configuration.ShardIDs,
		configuration.AutoSharded,
	)

	m.shardCount.Store(shardCount)

	ready, err := m.startShards(ctx, shardIDs, shardCount)
	if err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}

	<-ready

	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	m.shards.Range(func(_ int32, shard *Shard) bool {
		shard.Stop(ctx, websocket.StatusNormalClosure)

		return true
	})

	m.producer.Close()

	return nil
}

// getInitialShardCount returns the shard IDs and shard count for the manager.
func (m *Manager) getInitialShardCount(customShardCount int32, customShardIDs string, autoSharded bool) ([]int32, int32) {
	config := m.sandwich.config.Load()

	var shardCount int32

	var shardIDs []int32

	if autoSharded {
		shardCount = m.gateway.Load().Shards

		if customShardIDs == "" {
			for i := range shardCount {
				shardIDs = append(shardIDs, i)
			}
		} else {
			shardIDs = returnRangeInt32(config.Sandwich.NodeCount, config.Sandwich.NodeID, customShardIDs, shardCount)
		}
	} else {
		shardCount = customShardCount
		shardIDs = returnRangeInt32(config.Sandwich.NodeCount, config.Sandwich.NodeID, customShardIDs, shardCount)
	}

	// If we have a node count, split the shards evenly across nodes
	if config.Sandwich.NodeCount > 1 {
		filteredShardIDs := make([]int32, 0, len(shardIDs))

		// Only keep shards that belong to this node based on modulo
		for _, id := range shardIDs {
			if id%config.Sandwich.NodeCount == config.Sandwich.NodeID {
				filteredShardIDs = append(filteredShardIDs, id)
			}
		}

		shardIDs = filteredShardIDs
	}

	return shardIDs, shardCount
}

func (m *Manager) startShards(ctx context.Context, shardIDs []int32, shardCount int32) (ready chan struct{}, err error) {
	ready = make(chan struct{})

	now := time.Now()
	m.startedAt.Store(&now)

	m.shardCount.Store(shardCount)

	// If we have no shards, we can't start the manager
	if len(shardIDs) == 0 {
		return ready, ErrManagerMissingShards
	}

	// Kill any shards that are already running
	m.shards.Range(func(_ int32, shard *Shard) bool {
		shard.Stop(ctx, websocket.StatusNormalClosure)

		return true
	})

	// Create new shards
	for _, shardID := range shardIDs {
		shard := NewShard(m.sandwich, m, shardID)
		m.shards.Store(shardID, shard)
	}

	initialShard, ok := m.shards.Load(shardIDs[0])
	if !ok {
		panic("failed to load initial shard")
	}

	if err := initialShard.ConnectWithRetry(ctx); err != nil {
		return ready, fmt.Errorf("failed to connect to initial shard: %w", err)
	}

	go initialShard.Start(ctx)

	openWg := sync.WaitGroup{}

	for _, shardID := range shardIDs[1:] {
		shard, ok := m.shards.Load(shardID)
		if !ok {
			panic("failed to load shard")
		}

		openWg.Add(1)

		go func(shard *Shard) {
			defer openWg.Done()

			if err := shard.ConnectWithRetry(ctx); err != nil {
				return
			}

			go shard.Start(ctx)
		}(shard)
	}

	openWg.Wait()

	// All shards have now connected, but are not ready yet.

	go func() {
		m.shards.Range(func(_ int32, shard *Shard) bool {
			shard.waitForReady()

			return true
		})

		close(ready)
	}()

	return ready, nil
}
