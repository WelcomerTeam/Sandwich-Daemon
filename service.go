package sandwich

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/RealRock/bucketstore"
	"github.com/WelcomerTeam/RealRock/limiter"
	"github.com/WelcomerTeam/Sandwich-Daemon/pkg/syncmap"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

var Version = "2.0.0"

type Sandwich struct {
	logger *slog.Logger

	configProvider ConfigProvider
	config         *atomic.Pointer[Configuration]

	eventProvider    EventProvider
	identifyProvider IdentifyProvider
	producerProvider ProducerProvider
	stateProvider    StateProvider

	client *http.Client

	gatewayLimiter  *limiter.DurationLimiter
	identifyBuckets *bucketstore.BucketStore

	managers *syncmap.Map[string, *Manager]

	guildChunks *csmap.CsMap[discord.Snowflake, *GuildChunk]

	panicHandler PanicHandler
}

type PanicHandler func(sandwich *Sandwich, r any)

type GuildChunk struct {
	complete        *atomic.Bool
	chunkingChannel chan GuildChunkPartial
	startedAt       *atomic.Pointer[time.Time]
	completedAt     *atomic.Pointer[time.Time]
}

type GuildChunkPartial struct {
	chunkIndex int32
	chunkCount int32
	nonce      string
}

func NewSandwich(logger *slog.Logger, configProvider ConfigProvider, client *http.Client, eventProvider EventProvider, identifyProvider IdentifyProvider, producerProvider ProducerProvider, stateProvider StateProvider) *Sandwich {
	return &Sandwich{
		logger: logger,

		configProvider: configProvider,
		config:         &atomic.Pointer[Configuration]{},

		eventProvider:    eventProvider,
		identifyProvider: identifyProvider,
		producerProvider: producerProvider,
		stateProvider:    stateProvider,

		client: client,

		gatewayLimiter:  limiter.NewDurationLimiter(1, time.Second),
		identifyBuckets: bucketstore.NewBucketStore(),

		managers: &syncmap.Map[string, *Manager]{},

		guildChunks: csmap.Create[discord.Snowflake, *GuildChunk](),

		panicHandler: nil,
	}
}

func (sandwich *Sandwich) WithPanicHandler(panicHandler PanicHandler) *Sandwich {
	sandwich.panicHandler = panicHandler

	return sandwich
}

func (sandwich *Sandwich) Start(ctx context.Context) error {
	sandwich.logger.Info("Starting Sandwich")

	if err := sandwich.getConfig(ctx); err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if sandwich.client == nil {
		sandwich.client = http.DefaultClient
	}

	// TODO: setup GRPC
	// TODO: setup Prometheus
	// TODO: setup HTTP server

	sandwich.startManagers(ctx)

	return nil
}

func (sandwich *Sandwich) Stop(ctx context.Context) error {
	sandwich.logger.Info("Stopping Sandwich")

	sandwich.managers.Range(func(_ string, manager *Manager) bool {
		manager.Stop(ctx)

		return true
	})

	return nil
}

func (sandwich *Sandwich) getConfig(ctx context.Context) error {
	sandwich.logger.Debug("Getting config")

	config, err := sandwich.configProvider.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	sandwich.config.Store(config)

	// TODO: broadcast config change

	return nil
}

// startManagers starts all managers.
func (sandwich *Sandwich) startManagers(ctx context.Context) {
	sandwich.logger.Debug("Starting managers")

	managers := sandwich.config.Load().Managers

	for _, managerConfig := range managers {
		manager := NewManager(sandwich, managerConfig)
		sandwich.managers.Store(managerConfig.ApplicationIdentifier, manager)

		if err := sandwich.validateManagerConfig(managerConfig); err != nil {
			sandwich.logger.Error("Failed to validate manager config", "error", err)

			manager.SetStatus(ManagerStatusFailed)

			continue
		}

		if err := manager.Initialize(ctx); err != nil {
			sandwich.logger.Error("Failed to initialize manager", "error", err)

			manager.SetStatus(ManagerStatusFailed)

			continue
		}

		if manager.configuration.Load().AutoStart {
			go func(manager *Manager) {
				if err := manager.Start(ctx); err != nil {
					manager.SetStatus(ManagerStatusFailed)
				}
			}(manager)
		}
	}
}

// validateManagerConfig validates a manager configuration.
func (sandwich *Sandwich) validateManagerConfig(managerConfig *ManagerConfiguration) error {
	if managerConfig.ApplicationIdentifier == "" {
		return ErrManagerMissingIdentifier
	}

	if managerConfig.BotToken == "" {
		return ErrManagerMissingBotToken
	}

	if _, ok := sandwich.managers.Load(managerConfig.ApplicationIdentifier); ok {
		return ErrManagerIdentifierExists
	}

	return nil
}
