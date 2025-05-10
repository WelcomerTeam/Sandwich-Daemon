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

	identifier string

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

	status *atomic.Pointer[ManagerStatus]

	values map[string]any
}

func NewManager(s *Sandwich, config *ManagerConfiguration) *Manager {
	manager := &Manager{
		logger: s.logger.With("manager", config.ApplicationIdentifier),

		identifier: config.ApplicationIdentifier,

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

		status: &atomic.Pointer[ManagerStatus]{},

		values: config.Values,
	}

	manager.configuration.Store(config)

	manager.SetStatus(ManagerStatusIdle)

	return manager
}

func (manager *Manager) SetStatus(status ManagerStatus) {
	UpdateManagerStatus(manager.identifier, status)
	manager.status.Store(&status)
	manager.logger.Info("Manager status updated", "status", status.String())
}

func (manager *Manager) SetUser(user *discord.User) {
	existingUser := manager.user.Load()
	manager.user.Store(user)

	if existingUser != nil && existingUser.ID == user.ID {
		return
	}

	manager.logger.Debug("Manager user updated", "user", user.Username)

	configuration := manager.configuration.Load()

	manager.shards.Range(func(_ int32, shard *Shard) bool {
		shard.SetMetadata(configuration)

		return true
	})
}

// Initialize initializes the manager. This includes checking the gateway
func (manager *Manager) Initialize(ctx context.Context) error {
	manager.logger.Debug("Initializing manager")

	manager.sandwich.gatewayLimiter.Lock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discord.EndpointGatewayBot, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", manager.configuration.Load().BotToken))

	resp, err := manager.sandwich.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}

	defer resp.Body.Close()

	var gatewayBotResponse discord.GatewayBotResponse
	if err := json.NewDecoder(resp.Body).Decode(&gatewayBotResponse); err != nil {
		return fmt.Errorf("failed to decode gateway bot response: %w", err)
	}

	manager.gateway.Store(&gatewayBotResponse)
	manager.gatewaySessionStartLimitRemaining.Store(gatewayBotResponse.SessionStartLimit.Remaining)

	configuration := manager.configuration.Load()

	clientName := configuration.ClientName

	// If the client name includes a random suffix, we need to add a random suffix to the client name.
	if configuration.IncludeRandomSuffix {
		clientName = fmt.Sprintf("%s-%s", clientName, randomHex(8))
	}

	producer, err := manager.sandwich.producerProvider.GetProducer(ctx, configuration.ApplicationIdentifier, clientName)
	if err != nil {
		return fmt.Errorf("failed to get producer: %w", err)
	}

	manager.producer = producer

	manager.logger.Debug("Manager initialized")

	return nil
}

func (manager *Manager) Start(ctx context.Context) error {
	manager.logger.Info("Starting manager")

	manager.SetStatus(ManagerStatusStarting)

	configuration := manager.configuration.Load()

	shardIDs, shardCount := manager.getInitialShardCount(
		configuration.ShardCount,
		configuration.ShardIDs,
		configuration.AutoSharded,
	)

	manager.logger.Debug("Initializing shards", "shard_count", shardCount, "shard_ids", shardIDs)

	manager.shardCount.Store(shardCount)

	ready, err := manager.startShards(ctx, shardIDs, shardCount)
	if err != nil {
		manager.logger.Error("Failed to start shards", "error", err)

		manager.SetStatus(ManagerStatusFailed)

		return fmt.Errorf("failed to start: %w", err)
	}

	<-ready

	manager.SetStatus(ManagerStatusReady)

	return nil
}

func (manager *Manager) Stop(ctx context.Context) error {
	manager.SetStatus(ManagerStatusStopping)

	manager.shards.Range(func(_ int32, shard *Shard) bool {
		shard.Stop(ctx, websocket.StatusNormalClosure)

		return true
	})

	if manager.producer != nil {
		manager.producer.Close()
	}

	manager.SetStatus(ManagerStatusStopped)

	return nil
}

// getInitialShardCount returns the shard IDs and shard count for the manager.
func (manager *Manager) getInitialShardCount(customShardCount int32, customShardIDs string, autoSharded bool) ([]int32, int32) {
	config := manager.sandwich.config.Load()

	var shardCount int32

	var shardIDs []int32

	if autoSharded {
		shardCount = manager.gateway.Load().Shards

		if customShardIDs == "" {
			for i := range shardCount {
				shardIDs = append(shardIDs, i)
			}
		} else {
			shardIDs = returnRangeInt32(config.Sandwich.NodeCount, config.Sandwich.NodeID, customShardIDs, shardCount)
		}
	} else {
		shardCount = customShardCount

		if customShardIDs == "" {
			for i := range shardCount {
				shardIDs = append(shardIDs, i)
			}
		} else {
			shardIDs = returnRangeInt32(config.Sandwich.NodeCount, config.Sandwich.NodeID, customShardIDs, shardCount)
		}
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

func (manager *Manager) startShards(ctx context.Context, shardIDs []int32, shardCount int32) (ready chan struct{}, err error) {
	manager.logger.Info("Starting shards", "shard_count", shardCount, "shard_ids", shardIDs)

	ready = make(chan struct{})

	now := time.Now()
	manager.startedAt.Store(&now)

	manager.shardCount.Store(shardCount)

	// If we have no shards, we can't start the manager
	if len(shardIDs) == 0 {
		manager.logger.Error("No shards to start")

		return ready, ErrManagerMissingShards
	}

	// Kill any shards that are already running
	manager.shards.Range(func(_ int32, shard *Shard) bool {
		shard.Stop(ctx, websocket.StatusNormalClosure)

		return true
	})

	// Create new shards
	for _, shardID := range shardIDs {
		shard := NewShard(manager.sandwich, manager, shardID)

		manager.shards.Store(shardID, shard)
	}

	manager.SetStatus(ManagerStatusConnecting)

	initialShard, ok := manager.shards.Load(shardIDs[0])
	if !ok {
		panic("failed to load initial shard")
	}

	if err := initialShard.ConnectWithRetry(ctx); err != nil {
		manager.logger.Error("Failed to connect to initial shard", "error", err)

		return ready, fmt.Errorf("failed to connect to initial shard: %w", err)
	}

	go initialShard.Start(ctx)

	if err := initialShard.waitForReady(); err != nil {
		manager.logger.Error("Failed to wait for initial shard", "error", err)

		return ready, fmt.Errorf("failed to wait for initial shard: %w", err)
	}

	manager.logger.Debug("Initial shard connected", "shard_id", shardIDs[0])

	manager.SetStatus(ManagerStatusConnected)

	openWg := sync.WaitGroup{}

	for _, shardID := range shardIDs[1:] {
		shard, ok := manager.shards.Load(shardID)
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

	manager.logger.Debug("All shards connected")

	// All shards have now connected, but are not ready yet.

	go func() {
		manager.shards.Range(func(index int32, shard *Shard) bool {
			// Skip the initial shard
			if index == 0 {
				return true
			}

			shard.waitForReady()

			return true
		})

		close(ready)
	}()

	return ready, nil
}
