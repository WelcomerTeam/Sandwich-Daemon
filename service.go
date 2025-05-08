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
}

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
	}
}

func (s *Sandwich) Start(ctx context.Context) error {
	if err := s.getConfig(ctx); err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if s.client == nil {
		s.client = http.DefaultClient
	}

	// TODO: setup GRPC
	// TODO: setup Prometheus
	// TODO: setup HTTP server

	s.startManagers(ctx)

	return nil
}

func (s *Sandwich) Stop(ctx context.Context) error {
	s.managers.Range(func(_ string, manager *Manager) bool {
		manager.Stop(ctx)

		return true
	})

	return nil
}

func (s *Sandwich) getConfig(ctx context.Context) error {
	config, err := s.configProvider.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	s.config.Store(config)

	// TODO: broadcast config change

	return nil
}

// startManagers starts all managers.
func (s *Sandwich) startManagers(ctx context.Context) {
	managers := s.config.Load().Managers

	for _, managerConfig := range managers {
		manager := NewManager(s, managerConfig)
		s.managers.Store(managerConfig.ApplicationIdentifier, manager)

		if err := s.validateManagerConfig(managerConfig); err != nil {
			// TODO: set manager status to failed
			continue
		}

		if err := manager.Initialize(ctx); err != nil {
			// TODO: set manager status to failed
			continue
		}

		if manager.configuration.Load().AutoStart {
			go manager.Start(ctx)
		}
	}
}

// validateManagerConfig validates a manager configuration.
func (s *Sandwich) validateManagerConfig(managerConfig *ManagerConfiguration) error {
	if managerConfig.ApplicationIdentifier == "" {
		return ErrManagerMissingIdentifier
	}

	if managerConfig.BotToken == "" {
		return ErrManagerMissingBotToken
	}

	if _, ok := s.managers.Load(managerConfig.ApplicationIdentifier); ok {
		return ErrManagerIdentifierExists
	}

	return nil
}
