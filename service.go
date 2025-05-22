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
	csmap "github.com/mhmtszr/concurrent-swiss-map"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Version = "2.0.0-rc.4"

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

		listenerCounter: &atomic.Int32{},
		listeners:       &syncmap.Map[int32, chan *listenerData]{},
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
	sandwich.logger.Info("Starting Sandwich")

	if err := sandwich.getConfig(ctx); err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if sandwich.client == nil {
		sandwich.client = http.DefaultClient
	}

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
			slog.Info("Updated application configuration", "application_identifier", applicationConfig.ApplicationIdentifier)
			application.configuration.Store(applicationConfig)
		}
	}

	// TODO: broadcast config change

	sandwich.broadcast(SandwichEventConfigUpdate, nil)

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

		if _, err := sandwich.addApplication(ctx, applicationConfig); err != nil {
			sandwich.logger.Error("Failed to add application", "error", err)

			continue
		}
	}
}

func (sandwich *Sandwich) addApplication(ctx context.Context, applicationConfig *ApplicationConfiguration) (*Application, error) {
	application := NewApplication(sandwich, applicationConfig)
	sandwich.applications.Store(applicationConfig.ApplicationIdentifier, application)

	if err := application.Initialize(ctx); err != nil {
		sandwich.logger.Error("Failed to initialize application", "error", err)

		application.SetStatus(ApplicationStatusFailed)

		return application, err
	}

	if application.configuration.Load().AutoStart {
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

	if _, ok := sandwich.applications.Load(applicationConfig.ApplicationIdentifier); ok {
		return ErrApplicationIdentifierExists
	}

	return nil
}

type listenerData struct {
	timestamp time.Time
	payload   []byte
}

func (sandwich *Sandwich) addListener(listener chan *listenerData) int32 {
	counter := sandwich.listenerCounter.Add(1)

	sandwich.listeners.Store(counter, listener)

	return counter
}

func (sandwich *Sandwich) removeListener(counter int32) {
	sandwich.listeners.Delete(counter)
}

func (sandwich *Sandwich) broadcast(eventType string, data any) error {
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

// Conversions

func applicationToPB(application *Application) *pb.SandwichApplication {
	configuration := application.configuration.Load()

	var userID int64

	if applicationUser := application.user.Load(); applicationUser != nil {
		userID = int64(applicationUser.ID)
	}

	var valuesJSON []byte

	if configuration.Values != nil {
		valuesJSON, _ = json.Marshal(configuration.Values)
	}

	shards := make(map[int32]*pb.Shard)

	if application.shards != nil {
		application.shards.Range(func(shardIndex int32, shard *Shard) bool {
			shards[shardIndex] = &pb.Shard{
				Id:                shardIndex,
				Status:            shard.status.Load(),
				StartedAt:         shard.startedAt.Load().Unix(),
				UnavailableGuilds: int32(shard.unavailableGuilds.Count()),
				LazyGuilds:        int32(shard.lazyGuilds.Count()),
				Guilds:            int32(shard.guilds.Count()),
				Sequence:          shard.sequence.Load(),
				LastHeartbeatSent: shard.lastHeartbeatSent.Load().Unix(),
				LastHeartbeatAck:  shard.lastHeartbeatAck.Load().Unix(),
				GatewayLatency:    shard.gatewayLatency.Load(),
			}
			return true
		})
	}

	var startedAt int64

	if application.shards != nil && application.startedAt.Load() != nil {
		startedAt = application.startedAt.Load().Unix()
	}

	return &pb.SandwichApplication{
		ApplicationIdentifier: configuration.ApplicationIdentifier,
		ProducerIdentifier:    configuration.ProducerIdentifier,
		DisplayName:           configuration.DisplayName,
		BotToken:              configuration.BotToken,
		ShardCount:            application.shardCount.Load(),
		AutoSharded:           configuration.AutoSharded,
		Status:                application.status.Load(),
		StartedAt:             startedAt,
		UserId:                userID,
		Values:                valuesJSON,
		Shards:                shards,
	}
}

func UserToPB(user *discord.User) *pb.User {
	userPB := &pb.User{
		ID:            int64(user.ID),
		Username:      user.Username,
		Discriminator: user.Discriminator,
		GlobalName:    user.GlobalName,
		Avatar:        user.Avatar,
		Bot:           user.Bot,
		System:        user.System,
		MFAEnabled:    user.MFAEnabled,
		Banner:        user.Banner,
		AccentColour:  int32(user.AccentColor),
		Locale:        user.Locale,
		Verified:      user.Verified,
		Email:         user.Email,
		Flags:         int32(user.Flags),
		PremiumType:   int32(user.PremiumType),
		PublicFlags:   int32(user.PublicFlags),
		DMChannelID:   0,
	}

	if user.DMChannelID != nil {
		userPB.DMChannelID = int64(*user.DMChannelID)
	}

	return userPB
}

func snowflakeListToInt64List(snowflakes []discord.Snowflake) []int64 {
	int64List := make([]int64, len(snowflakes))

	for i, snowflake := range snowflakes {
		int64List[i] = int64(snowflake)
	}

	return int64List
}

func GuildMemberToPB(guildMember *discord.GuildMember) *pb.GuildMember {
	guildMemberPB := &pb.GuildMember{
		User:                       nil,
		GuildID:                    0,
		Nick:                       guildMember.Nick,
		Avatar:                     guildMember.Avatar,
		Roles:                      snowflakeListToInt64List(guildMember.Roles),
		JoinedAt:                   guildMember.JoinedAt.Format(time.RFC3339),
		PremiumSince:               guildMember.PremiumSince,
		Deaf:                       guildMember.Deaf,
		Mute:                       guildMember.Mute,
		Pending:                    guildMember.Pending,
		Permissions:                0,
		CommunicationDisabledUntil: guildMember.CommunicationDisabledUntil,
	}

	if guildMember.User != nil {
		guildMemberPB.User = UserToPB(guildMember.User)
	}

	if guildMember.GuildID != nil {
		guildMemberPB.GuildID = int64(*guildMember.GuildID)
	}

	if guildMember.Permissions != nil {
		guildMemberPB.Permissions = int64(*guildMember.Permissions)
	}

	return guildMemberPB
}
