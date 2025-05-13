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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	applications *syncmap.Map[string, *Application]

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

		applications: &syncmap.Map[string, *Application]{},

		guildChunks: csmap.Create[discord.Snowflake, *GuildChunk](),

		panicHandler: nil,
	}
}

func (sandwich *Sandwich) WithPanicHandler(panicHandler PanicHandler) *Sandwich {
	sandwich.panicHandler = panicHandler

	return sandwich
}

func (sandwich *Sandwich) WithPrometheusAnalytics(
	server *http.Server,
	registry *prometheus.Registry,
	opts promhttp.HandlerOpts,
) *Sandwich {
	if registry == nil {
		registry = prometheus.NewPedanticRegistry()
	}

	registry.MustRegister(
		EventMetrics.EventsTotal,
		EventMetrics.GatewayLatency,

		ShardMetrics.ApplicationStatus,
		ShardMetrics.ShardStatus,

		StateMetrics.Channels,
		StateMetrics.Emojis,
		StateMetrics.GuildMembers,
		StateMetrics.GuildRoles,
		StateMetrics.Guilds,
		StateMetrics.Stickers,
		StateMetrics.UnavailableGuilds,
		StateMetrics.Users,
		StateMetrics.VoiceStates,
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, opts))

	server.Handler = mux

	go func() {
		slog.Info("Starting Prometheus HTTP server", "host", server.Addr)

		var err error

		if server.TLSConfig != nil {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}

		if err != nil {
			panic(fmt.Errorf("failed to start Prometheus HTTP server: %w", err))
		}
	}()

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
	// TODO: setup HTTP server

	sandwich.startApplications(ctx)

	return nil
}

func (sandwich *Sandwich) Stop(ctx context.Context) {
	sandwich.logger.Info("Stopping Sandwich")

	sandwich.applications.Range(func(_ string, application *Application) bool {
		application.Stop(ctx)

		return true
	})
}

func (sandwich *Sandwich) getConfig(ctx context.Context) error {
	sandwich.logger.Debug("Getting config")

	config, err := sandwich.configProvider.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	sandwich.config.Store(config)

	// Update application configurations
	for _, applicationConfig := range config.Applications {
		if application, ok := sandwich.applications.Load(applicationConfig.ApplicationIdentifier); ok {
			application.configuration.Store(applicationConfig)
			slog.Info("Updated application configuration", "application_identifier", applicationConfig.ApplicationIdentifier)
		}
	}

	// TODO: broadcast config change

	return nil
}

// startApplications starts all applications.
func (sandwich *Sandwich) startApplications(ctx context.Context) {
	sandwich.logger.Debug("Starting applications")

	applications := sandwich.config.Load().Applications

	for _, applicationConfig := range applications {
		if err := sandwich.validateApplicationConfig(applicationConfig); err != nil {
			sandwich.logger.Error("Failed to validate application config", "error", err)

			continue
		}

		application := NewApplication(sandwich, applicationConfig)
		sandwich.applications.Store(applicationConfig.ApplicationIdentifier, application)

		if err := application.Initialize(ctx); err != nil {
			sandwich.logger.Error("Failed to initialize application", "error", err)

			application.SetStatus(ApplicationStatusFailed)

			continue
		}

		if application.configuration.Load().AutoStart {
			go func(application *Application) {
				if err := application.Start(ctx); err != nil {
					application.SetStatus(ApplicationStatusFailed)
				}
			}(application)
		}
	}
}

// validateApplicationConfig validates a application configuration.
func (sandwich *Sandwich) validateApplicationConfig(applicationConfig *ApplicationConfiguration) error {
	if applicationConfig.ApplicationIdentifier == "" {
		return ErrApplicationMissingIdentifier
	}

	if applicationConfig.BotToken == "" {
		return ErrApplicationMissingBotToken
	}

	if _, ok := sandwich.applications.Load(applicationConfig.ApplicationIdentifier); ok {
		return ErrApplicationIdentifierExists
	}

	return nil
}
