package sandwich

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/Sandwich-Daemon/pkg/bucketstore"
	"github.com/WelcomerTeam/Sandwich-Daemon/pkg/limiter"
	"github.com/WelcomerTeam/Sandwich-Daemon/pkg/syncmap"
	pb "github.com/WelcomerTeam/Sandwich-Daemon/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Version = "2.3.1"

type Sandwich struct {
	Logger *slog.Logger

	configProvider ConfigProvider
	Config         *atomic.Pointer[Configuration]

	eventProvider    EventProvider
	identifyProvider IdentifyProvider
	producerProvider ProducerProvider
	stateProvider    StateProvider
	dedupeProvider   DedupeProvider

	Client *http.Client

	gatewayLimiter  *limiter.DurationLimiter
	identifyBuckets *bucketstore.BucketStore

	Applications *syncmap.Map[string, *Application]

	guildChunks *syncmap.Map[discord.Snowflake, *GuildChunk]

	panicHandler PanicHandler

	listenerCounter *atomic.Int32
	listeners       *syncmap.Map[int32, chan *listenerData]
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

func NewSandwich(logger *slog.Logger, configProvider ConfigProvider, client *http.Client, eventProvider EventProvider, identifyProvider IdentifyProvider, producerProvider ProducerProvider, stateProvider StateProvider, dedupeProvider DedupeProvider) *Sandwich {
	sandwich := &Sandwich{
		Logger: logger,

		configProvider: configProvider,
		Config:         &atomic.Pointer[Configuration]{},

		eventProvider:    eventProvider,
		identifyProvider: identifyProvider,
		producerProvider: producerProvider,
		stateProvider:    stateProvider,
		dedupeProvider:   dedupeProvider,

		Client: client,

		gatewayLimiter:  limiter.NewDurationLimiter(1, time.Second),
		identifyBuckets: bucketstore.NewBucketStore(),

		Applications: &syncmap.Map[string, *Application]{},

		guildChunks: &syncmap.Map[discord.Snowflake, *GuildChunk]{},

		panicHandler: nil,

		listenerCounter: &atomic.Int32{},
		listeners:       &syncmap.Map[int32, chan *listenerData]{},
	}

	// Start background cleanup for completed guild chunks
	go sandwich.cleanupGuildChunks(context.Background())

	return sandwich
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

		StateMetrics.StateRequests,
		StateMetrics.StateHits,
		StateMetrics.StateMisses,
		StateMetrics.Channels,
		StateMetrics.Emojis,
		StateMetrics.GuildMembers,
		StateMetrics.GuildRoles,
		StateMetrics.Guilds,
		StateMetrics.Stickers,
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

func (sandwich *Sandwich) WithGRPCServer(listenerConfig *net.ListenConfig, network, address string, server *grpc.Server) *Sandwich {
	pb.RegisterSandwichServer(server, sandwich.NewGRPCServer())

	// Enables server reflection
	reflection.Register(server)

	go func() {
		slog.Info("Starting GRPC server", "network", network, "host", address)

		var err error

		var listener net.Listener

		if listenerConfig != nil {
			listener, err = listenerConfig.Listen(context.Background(), network, address)
		} else {
			listener, err = net.Listen(network, address)
		}

		if err == nil {
			err = server.Serve(listener)
		}

		if err != nil {
			panic(fmt.Errorf("failed to start GRPC server: %w", err))
		}
	}()

	return sandwich
}

func (sandwich *Sandwich) Start(ctx context.Context) error {
	sandwich.Logger.Info("Starting Sandwich")

	if err := sandwich.getConfig(ctx); err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if sandwich.Client == nil {
		sandwich.Client = http.DefaultClient
	}

	sandwich.startApplications(ctx)

	return nil
}

func (sandwich *Sandwich) Stop(ctx context.Context) {
	sandwich.Logger.Info("Stopping Sandwich")

	sandwich.Applications.Range(func(_ string, application *Application) bool {
		application.Stop(ctx)

		return true
	})
}

func (sandwich *Sandwich) getConfig(ctx context.Context) error {
	sandwich.Logger.Debug("Getting config")

	config, err := sandwich.configProvider.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	sandwich.Config.Store(config)

	// Update application configurations
	for _, applicationConfig := range config.Applications {
		if application, ok := sandwich.Applications.Load(applicationConfig.ApplicationIdentifier); ok {
			slog.Info("Updated application configuration", "application_identifier", applicationConfig.ApplicationIdentifier)
			application.Configuration.Store(applicationConfig)
		}
	}

	// TODO: broadcast config change

	sandwich.Broadcast(SandwichEventConfigUpdate, nil)

	return nil
}

// startApplications starts all applications.
func (sandwich *Sandwich) startApplications(ctx context.Context) {
	sandwich.Logger.Debug("Starting applications")

	applications := sandwich.Config.Load().Applications

	for _, applicationConfig := range applications {
		if err := sandwich.validateApplicationConfig(applicationConfig); err != nil {
			sandwich.Logger.Error("Failed to validate application config", "error", err)

			continue
		}

		if _, err := sandwich.AddApplication(ctx, applicationConfig); err != nil {
			sandwich.Logger.Error("Failed to add application", "error", err)

			continue
		}
	}
}

func (sandwich *Sandwich) AddApplication(ctx context.Context, applicationConfig *ApplicationConfiguration) (*Application, error) {
	application := NewApplication(sandwich, applicationConfig)
	sandwich.Applications.Store(applicationConfig.ApplicationIdentifier, application)

	if err := application.Initialize(ctx); err != nil {
		sandwich.Logger.Error("Failed to initialize application", "error", err)

		application.SetStatus(ApplicationStatusFailed)

		return application, err
	}

	if application.Configuration.Load().AutoStart {
		go func(application *Application) {
			if err := application.Start(ctx); err != nil {
				application.SetStatus(ApplicationStatusFailed)
			}
		}(application)
	}

	return application, nil
}

// validateApplicationConfig validates a application configuration.
func (sandwich *Sandwich) validateApplicationConfig(applicationConfig *ApplicationConfiguration) error {
	if applicationConfig.ApplicationIdentifier == "" {
		return ErrApplicationMissingIdentifier
	}

	if applicationConfig.BotToken == "" {
		return ErrApplicationMissingBotToken
	}

	if _, ok := sandwich.Applications.Load(applicationConfig.ApplicationIdentifier); ok {
		return ErrApplicationIdentifierExists
	}

	return nil
}

type listenerData struct {
	timestamp time.Time
	payload   []byte
}

func (sandwich *Sandwich) AddListener(listener chan *listenerData) int32 {
	counter := sandwich.listenerCounter.Add(1)

	sandwich.listeners.Store(counter, listener)

	return counter
}

func (sandwich *Sandwich) RemoveListener(counter int32) {
	sandwich.listeners.Delete(counter)
}

func (sandwich *Sandwich) Broadcast(eventType string, data any) error {
	payloadDataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal payload data: %w", err)
	}

	payload := ProducedPayload{
		GatewayPayload: discord.GatewayPayload{
			Op:   discord.GatewayOpDispatch,
			Type: eventType,
			Data: payloadDataBytes,
		},
		Extra:    nil,
		Metadata: ProducedMetadata{},
		Trace:    Trace{},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	listenData := &listenerData{
		timestamp: time.Now(),
		payload:   payloadBytes,
	}

	sandwich.listeners.Range(func(_ int32, listener chan *listenerData) bool {
		listener <- listenData

		return true
	})

	return nil
}

// Custom conversions

func applicationToPB(application *Application) *pb.SandwichApplication {
	configuration := application.Configuration.Load()

	var userID int64

	if applicationUser := application.User.Load(); applicationUser != nil {
		userID = int64(applicationUser.ID)
	}

	var valuesJSON []byte

	if configuration.Values != nil {
		valuesJSON, _ = json.Marshal(configuration.Values)
	}

	shards := make(map[int32]*pb.Shard)

	if application.Shards != nil {
		application.Shards.Range(func(shardIndex int32, shard *Shard) bool {
			shardPb := &pb.Shard{
				Id:                shardIndex,
				Status:            shard.Status.Load(),
				UnavailableGuilds: int32(shard.UnavailableGuilds.Count()),
				LazyGuilds:        int32(shard.LazyGuilds.Count()),
				Guilds:            int32(shard.Guilds.Count()),
				Sequence:          shard.sequence.Load(),
				GatewayLatency:    shard.GatewayLatency.Load(),
			}

			startedAt := shard.StartedAt.Load()
			if startedAt != nil {
				shardPb.StartedAt = startedAt.Unix()
			}

			lastHeartbeatSent := shard.LastHeartbeatSent.Load()
			if lastHeartbeatSent != nil {
				shardPb.LastHeartbeatSent = lastHeartbeatSent.Unix()
			}

			lastHeartbeatAck := shard.LastHeartbeatAck.Load()
			if lastHeartbeatAck != nil {
				shardPb.LastHeartbeatAck = lastHeartbeatAck.Unix()
			}

			shards[shardIndex] = shardPb

			return true
		})
	}

	var startedAt int64

	if application.Shards != nil && application.startedAt.Load() != nil {
		startedAt = application.startedAt.Load().Unix()
	}

	return &pb.SandwichApplication{
		ApplicationIdentifier: configuration.ApplicationIdentifier,
		ProducerIdentifier:    configuration.ProducerIdentifier,
		DisplayName:           configuration.DisplayName,
		BotToken:              configuration.BotToken,
		ShardCount:            application.ShardCount.Load(),
		AutoSharded:           configuration.AutoSharded,
		Status:                application.Status.Load(),
		StartedAt:             startedAt,
		UserId:                userID,
		Values:                valuesJSON,
		Shards:                shards,
	}
}

// cleanupGuildChunks periodically removes completed guild chunks to prevent memory leaks
func (sandwich *Sandwich) cleanupGuildChunks(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			var toDelete []discord.Snowflake

			// Find chunks that have been completed for more than 10 minutes
			sandwich.guildChunks.Range(func(guildID discord.Snowflake, chunk *GuildChunk) bool {
				if chunk.complete.Load() {
					completedAt := chunk.completedAt.Load()
					if completedAt != nil && now.Sub(*completedAt) > 10*time.Minute {
						toDelete = append(toDelete, guildID)
					}
				}
				return true
			})

			// Delete old completed chunks
			for _, guildID := range toDelete {
				sandwich.guildChunks.Delete(guildID)
			}

			if len(toDelete) > 0 {
				sandwich.Logger.Debug("Cleaned up completed guild chunks", "count", len(toDelete))
			}
		}
	}
}
