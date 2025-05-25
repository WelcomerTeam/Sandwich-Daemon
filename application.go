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
	Logger *slog.Logger

	Identifier string

	Sandwich      *Sandwich
	Configuration *atomic.Pointer[ApplicationConfiguration]

	Gateway                           *atomic.Pointer[discord.GatewayBotResponse]
	gatewaySessionStartLimitRemaining *atomic.Int32

	User *atomic.Pointer[discord.User]

	producer Producer

	ShardCount *atomic.Int32

	ready   chan struct{}
	readyWg sync.WaitGroup

	Shards *syncmap.Map[int32, *Shard]
	guilds *csmap.CsMap[discord.Snowflake, bool]

	startedAt *atomic.Pointer[time.Time]

	Status *atomic.Int32
}

func NewApplication(sandwich *Sandwich, config *ApplicationConfiguration) *Application {
	application := &Application{
		Logger: sandwich.Logger.With("application_identifier", config.ApplicationIdentifier),

		Identifier: config.ApplicationIdentifier,

		Sandwich:      sandwich,
		Configuration: &atomic.Pointer[ApplicationConfiguration]{},

		Gateway:                           &atomic.Pointer[discord.GatewayBotResponse]{},
		gatewaySessionStartLimitRemaining: &atomic.Int32{},

		User: &atomic.Pointer[discord.User]{},

		producer: nil,

		ShardCount: &atomic.Int32{},

		ready:   make(chan struct{}),
		readyWg: sync.WaitGroup{},

		Shards: &syncmap.Map[int32, *Shard]{},
		guilds: csmap.Create[discord.Snowflake, bool](),

		startedAt: &atomic.Pointer[time.Time]{},

		Status: &atomic.Int32{},
	}

	application.Configuration.Store(config)

	application.SetStatus(ApplicationStatusIdle)

	return application
}

func (application *Application) SetStatus(status ApplicationStatus) {
	UpdateApplicationStatus(application.Identifier, status)
	application.Status.Store(int32(status))
	application.Logger.Info("Application status updated", "status", status.String())

	err := application.Sandwich.broadcast(SandwichApplicationStatusUpdate, ApplicationStatusUpdateEvent{
		Identifier: application.Identifier,
		Status:     status,
	})
	if err != nil {
		application.Logger.Error("Failed to broadcast application status update", "error", err)
	}
}

func (application *Application) SetUser(user *discord.User) {
	existingUser := application.User.Load()
	application.User.Store(user)

	if existingUser != nil && existingUser.ID == user.ID {
		return
	}

	application.Logger.Debug("Application user updated", "user", user.Username)

	configuration := application.Configuration.Load()

	application.Shards.Range(func(_ int32, shard *Shard) bool {
		shard.SetMetadata(configuration)

		return true
	})
}

// Initialize initializes the application. This includes checking the gateway
func (application *Application) Initialize(ctx context.Context) error {
	application.Logger.Debug("Initializing application")

	application.Sandwich.gatewayLimiter.Lock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discord.EndpointGatewayBot, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+application.Configuration.Load().BotToken)

	resp, err := application.Sandwich.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}

	defer resp.Body.Close()

	var gatewayBotResponse discord.GatewayBotResponse
	if err := json.NewDecoder(resp.Body).Decode(&gatewayBotResponse); err != nil {
		return fmt.Errorf("failed to decode gateway bot response: %w", err)
	}

	application.Gateway.Store(&gatewayBotResponse)
	application.gatewaySessionStartLimitRemaining.Store(gatewayBotResponse.SessionStartLimit.Remaining)

	configuration := application.Configuration.Load()

	clientName := configuration.ClientName

	// If the client name includes a random suffix, we need to add a random suffix to the client name.
	if configuration.IncludeRandomSuffix {
		clientName = fmt.Sprintf("%s-%s", clientName, randomHex(8))
	}

	producer, err := application.Sandwich.producerProvider.GetProducer(ctx, configuration.ApplicationIdentifier, clientName)
	if err != nil {
		return fmt.Errorf("failed to get producer: %w", err)
	}

	application.producer = producer

	application.Logger.Debug("Application initialized")

	return nil
}

func (application *Application) Start(ctx context.Context) error {
	application.Logger.Info("Starting application")

	application.SetStatus(ApplicationStatusStarting)

	configuration := application.Configuration.Load()

	shardIDs, shardCount := application.getInitialShardCount(
		configuration.ShardCount,
		configuration.ShardIDs,
		configuration.AutoSharded,
	)

	application.Logger.Debug("Initializing shards", "shard_count", shardCount, "shard_ids", shardIDs)

	application.ShardCount.Store(shardCount)

	ready, err := application.startShards(ctx, shardIDs, shardCount)
	if err != nil {
		application.Logger.Error("Failed to start shards", "error", err)

		application.SetStatus(ApplicationStatusFailed)

		return fmt.Errorf("failed to start: %w", err)
	}

	<-ready

	application.SetStatus(ApplicationStatusReady)

	return nil
}

func (application *Application) Stop(ctx context.Context) error {
	application.SetStatus(ApplicationStatusStopping)

	application.Shards.Range(func(_ int32, shard *Shard) bool {
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
	config := application.Sandwich.Config.Load()

	var shardCount int32

	var shardIDs []int32

	if autoSharded {
		shardCount = application.Gateway.Load().Shards

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

func (application *Application) startShards(ctx context.Context, shardIDs []int32, shardCount int32) (ready chan struct{}, err error) {
	application.Logger.Info("Starting shards", "shard_count", shardCount, "shard_ids", shardIDs)

	ready = make(chan struct{})

	now := time.Now()
	application.startedAt.Store(&now)

	application.ShardCount.Store(shardCount)

	// If we have no shards, we can't start the application
	if len(shardIDs) == 0 {
		application.Logger.Error("No shards to start")

		return ready, ErrApplicationMissingShards
	}

	// Kill any shards that are already running
	application.Shards.Range(func(_ int32, shard *Shard) bool {
		shard.Stop(ctx, websocket.StatusNormalClosure)

		return true
	})

	// Create new shards
	for _, shardID := range shardIDs {
		shard := NewShard(application.Sandwich, application, shardID)

		application.Shards.Store(shardID, shard)
	}

	application.SetStatus(ApplicationStatusConnecting)

	initialShard, ok := application.Shards.Load(shardIDs[0])
	if !ok {
		panic("failed to load initial shard")
	}

	if err := initialShard.ConnectWithRetry(ctx); err != nil {
		application.Logger.Error("Failed to connect to initial shard", "error", err)

		return ready, fmt.Errorf("failed to connect to initial shard: %w", err)
	}

	go initialShard.Start(ctx)

	if err := initialShard.waitForReady(); err != nil {
		application.Logger.Error("Failed to wait for initial shard", "error", err)

		return ready, fmt.Errorf("failed to wait for initial shard: %w", err)
	}

	application.Logger.Debug("Initial shard connected", "shard_id", shardIDs[0])

	application.SetStatus(ApplicationStatusConnected)

	openWg := sync.WaitGroup{}

	for _, shardID := range shardIDs[1:] {
		shard, ok := application.Shards.Load(shardID)
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

	application.Logger.Debug("All shards connected")

	// All shards have now connected, but are not ready yet.

	go func() {
		application.Shards.Range(func(index int32, shard *Shard) bool {
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
