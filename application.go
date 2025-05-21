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

type Application struct {
	logger *slog.Logger

	identifier string

	sandwich      *Sandwich
	configuration *atomic.Pointer[ApplicationConfiguration]

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

	status *atomic.Int32
}

func NewApplication(sandwich *Sandwich, config *ApplicationConfiguration) *Application {
	application := &Application{
		logger: sandwich.logger.With("application_identifier", config.ApplicationIdentifier),

		identifier: config.ApplicationIdentifier,

		sandwich:      sandwich,
		configuration: &atomic.Pointer[ApplicationConfiguration]{},

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

		status: &atomic.Int32{},
	}

	application.configuration.Store(config)

	application.SetStatus(ApplicationStatusIdle)

	return application
}

func (application *Application) SetStatus(status ApplicationStatus) {
	UpdateApplicationStatus(application.identifier, status)
	application.status.Store(int32(status))
	application.logger.Info("Application status updated", "status", status.String())

	err := application.sandwich.broadcast(SandwichApplicationStatusUpdate, ApplicationStatusUpdateEvent{
		Identifier: application.identifier,
		Status:     status,
	})
	if err != nil {
		application.logger.Error("Failed to broadcast application status update", "error", err)
	}
}

func (application *Application) SetUser(user *discord.User) {
	existingUser := application.user.Load()
	application.user.Store(user)

	if existingUser != nil && existingUser.ID == user.ID {
		return
	}

	application.logger.Debug("Application user updated", "user", user.Username)

	configuration := application.configuration.Load()

	application.shards.Range(func(_ int32, shard *Shard) bool {
		shard.SetMetadata(configuration)

		return true
	})
}

// Initialize initializes the application. This includes checking the gateway
func (application *Application) Initialize(ctx context.Context) error {
	application.logger.Debug("Initializing application")

	application.sandwich.gatewayLimiter.Lock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discord.EndpointGatewayBot, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+application.configuration.Load().BotToken)

	resp, err := application.sandwich.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}

	defer resp.Body.Close()

	var gatewayBotResponse discord.GatewayBotResponse
	if err := json.NewDecoder(resp.Body).Decode(&gatewayBotResponse); err != nil {
		return fmt.Errorf("failed to decode gateway bot response: %w", err)
	}

	application.gateway.Store(&gatewayBotResponse)
	application.gatewaySessionStartLimitRemaining.Store(gatewayBotResponse.SessionStartLimit.Remaining)

	configuration := application.configuration.Load()

	clientName := configuration.ClientName

	// If the client name includes a random suffix, we need to add a random suffix to the client name.
	if configuration.IncludeRandomSuffix {
		clientName = fmt.Sprintf("%s-%s", clientName, randomHex(8))
	}

	producer, err := application.sandwich.producerProvider.GetProducer(ctx, configuration.ApplicationIdentifier, clientName)
	if err != nil {
		return fmt.Errorf("failed to get producer: %w", err)
	}

	application.producer = producer

	application.logger.Debug("Application initialized")

	return nil
}

func (application *Application) Start(ctx context.Context) error {
	application.logger.Info("Starting application")

	application.SetStatus(ApplicationStatusStarting)

	configuration := application.configuration.Load()

	shardIDs, shardCount := application.getInitialShardCount(
		configuration.ShardCount,
		configuration.ShardIDs,
		configuration.AutoSharded,
	)

	application.logger.Debug("Initializing shards", "shard_count", shardCount, "shard_ids", shardIDs)

	application.shardCount.Store(shardCount)

	ready, err := application.startShards(ctx, shardIDs, shardCount)
	if err != nil {
		application.logger.Error("Failed to start shards", "error", err)

		application.SetStatus(ApplicationStatusFailed)

		return fmt.Errorf("failed to start: %w", err)
	}

	<-ready

	application.SetStatus(ApplicationStatusReady)

	return nil
}

func (application *Application) Stop(ctx context.Context) error {
	application.SetStatus(ApplicationStatusStopping)

	application.shards.Range(func(_ int32, shard *Shard) bool {
		shard.Stop(ctx, websocket.StatusNormalClosure)

		return true
	})

	if application.producer != nil {
		application.producer.Close()
	}

	application.SetStatus(ApplicationStatusStopped)

	return nil
}

// getInitialShardCount returns the shard IDs and shard count for the application.
func (application *Application) getInitialShardCount(customShardCount int32, customShardIDs string, autoSharded bool) ([]int32, int32) {
	config := application.sandwich.config.Load()

	var shardCount int32

	var shardIDs []int32

	if autoSharded {
		shardCount = application.gateway.Load().Shards

		if customShardIDs == "" {
			for i := int32(0); i < shardCount; i++ {
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

func (application *Application) startShards(ctx context.Context, shardIDs []int32, shardCount int32) (ready chan struct{}, err error) {
	application.logger.Info("Starting shards", "shard_count", shardCount, "shard_ids", shardIDs)

	ready = make(chan struct{})

	now := time.Now()
	application.startedAt.Store(&now)

	application.shardCount.Store(shardCount)

	// If we have no shards, we can't start the application
	if len(shardIDs) == 0 {
		application.logger.Error("No shards to start")

		return ready, ErrApplicationMissingShards
	}

	// Kill any shards that are already running
	application.shards.Range(func(_ int32, shard *Shard) bool {
		shard.Stop(ctx, websocket.StatusNormalClosure)

		return true
	})

	// Create new shards
	for _, shardID := range shardIDs {
		shard := NewShard(application.sandwich, application, shardID)

		application.shards.Store(shardID, shard)
	}

	application.SetStatus(ApplicationStatusConnecting)

	initialShard, ok := application.shards.Load(shardIDs[0])
	if !ok {
		panic("failed to load initial shard")
	}

	if err := initialShard.ConnectWithRetry(ctx); err != nil {
		application.logger.Error("Failed to connect to initial shard", "error", err)

		return ready, fmt.Errorf("failed to connect to initial shard: %w", err)
	}

	go initialShard.Start(ctx)

	if err := initialShard.waitForReady(); err != nil {
		application.logger.Error("Failed to wait for initial shard", "error", err)

		return ready, fmt.Errorf("failed to wait for initial shard: %w", err)
	}

	application.logger.Debug("Initial shard connected", "shard_id", shardIDs[0])

	application.SetStatus(ApplicationStatusConnected)

	openWg := sync.WaitGroup{}

	for _, shardID := range shardIDs[1:] {
		shard, ok := application.shards.Load(shardID)
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

	application.logger.Debug("All shards connected")

	// All shards have now connected, but are not ready yet.

	go func() {
		application.shards.Range(func(index int32, shard *Shard) bool {
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
