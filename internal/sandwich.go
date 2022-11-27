package internal

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/RealRock/bucketstore"
	"github.com/WelcomerTeam/RealRock/interfacecache"
	limiter "github.com/WelcomerTeam/RealRock/limiter"
	grpcServer "github.com/WelcomerTeam/Sandwich-Daemon/protobuf"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"github.com/fasthttp/session/v2"
	memory "github.com/fasthttp/session/v2/providers/memory"
	jsoniter "github.com/json-iterator/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
	"go.uber.org/atomic"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
)

// VERSION follows semantic versioning.
const VERSION = "1.8"

const (
	PermissionsDefault = 0o744
	PermissionWrite    = 0o600

	prometheusGatherInterval = 10 * time.Second
	cacheEjectorInterval     = 1 * time.Minute

	// Time to keep a member dedupe event for.
	memberDedupeExpiration = 1 * time.Minute
	memberDMExpiration     = 30 * time.Minute
)

var baseURL = url.URL{
	Scheme: "https",
	Host:   "discord.com",
}

var gatewayURL = url.URL{
	Scheme:   "wss",
	Host:     "gateway.discord.gg",
	RawQuery: "v=9&encoding=json",
}

type Sandwich struct {
	sync.Mutex

	ConfigurationLocation string `json:"configuration_location"`

	ctx    context.Context
	cancel func()

	Logger    zerolog.Logger `json:"-"`
	StartTime time.Time      `json:"start_time" yaml:"start_time"`

	configurationMu sync.RWMutex
	Configuration   *SandwichConfiguration `json:"configuration" yaml:"configuration"`

	Options SandwichOptions `json:"options" yaml:"options"`

	gatewayLimiter limiter.DurationLimiter

	ProducerClient *MQClient `json:"-"`

	IdentifyBuckets *bucketstore.BucketStore `json:"-"`

	EventsInflight *atomic.Int32 `json:"-"`

	managersMu sync.RWMutex
	Managers   map[string]*Manager `json:"managers" yaml:"managers"`

	globalPoolMu          sync.RWMutex
	globalPool            map[int32]chan []byte
	globalPoolAccumulator *atomic.Int32

	dedupeMu sync.RWMutex
	Dedupe   map[string]int64

	State  *SandwichState `json:"-"`
	Client *Client        `json:"-"`

	webhookBuckets *bucketstore.BucketStore
	statusCache    *interfacecache.InterfaceCache

	SessionProvider *session.Session

	RouterHandler fasthttp.RequestHandler `json:"-"`
	DistHandler   fasthttp.RequestHandler `json:"-"`

	// SandwichPayload pool
	payloadPool sync.Pool
	// ReceivedPayload pool
	receivedPool sync.Pool
	// SentPayload pool
	sentPool sync.Pool
}

// SandwichConfiguration represents the configuration file.
type SandwichConfiguration struct {
	Identify struct {
		// URL allows for variables:
		// {shard_id}, {shard_count}, {token} {token_hash}, {max_concurrency}
		URL string `json:"url" yaml:"url"`

		Headers map[string]string `json:"headers" yaml:"headers"`
	} `json:"identify" yaml:"identify"`

	Producer struct {
		Type          string                 `json:"type" yaml:"type"`
		Configuration map[string]interface{} `json:"configuration" yaml:"configuration"`
	} `json:"producer" yaml:"producer"`

	HTTP struct {
		// OAuth config used to identification.
		OAuth *oauth2.Config `json:"oauth" yaml:"oauth"`

		// List of discord user IDs that can access the dashboard.
		UserAccess []string `json:"user_access" yaml:"user_access"`
	} `json:"http" yaml:"http"`

	Webhooks []string `json:"webhooks" yaml:"webhooks"`

	Managers []*ManagerConfiguration `json:"managers" yaml:"managers"`
}

// SandwichOptions represents any options passable when creating
// the sandwich service
type SandwichOptions struct {
	ConfigurationLocation string  `json:"configuration_location" yaml:"configuration_location"`
	PrometheusAddress     string  `json:"prometheus_address" yaml:"prometheus_address"`
	GatewayURL            url.URL `json:"gateway_url" yaml:"gateway_url"`
	// BaseURL to send HTTP requests to. If empty, will use https://discord.com
	BaseURL url.URL `json:"base_url" yaml:"base_url"`

	GRPCNetwork            string `json:"grpc_network" yaml:"grpc_network"`
	GRPCHost               string `json:"grpc_host" yaml:"grpc_host"`
	GRPCCertFile           string `json:"grpc_cert_file" yaml:"grpc_cert_file"`
	GRPCServerNameOverride string `json:"grpc_server_name_override" yaml:"grpc_server_name_override"`

	HTTPHost    string `json:"http_host" yaml:"http_host"`
	HTTPEnabled bool   `json:"http_enabled" yaml:"http_enabled"`
}

// NewSandwich creates the application state and initializes it.
func NewSandwich(logger io.Writer, options SandwichOptions) (sg *Sandwich, err error) {
	sg = &Sandwich{
		Logger: zerolog.New(logger).With().Timestamp().Logger(),

		ConfigurationLocation: options.ConfigurationLocation,

		configurationMu: sync.RWMutex{},
		Configuration:   &SandwichConfiguration{},

		Options: options,

		gatewayLimiter: *limiter.NewDurationLimiter(1, time.Second),

		managersMu: sync.RWMutex{},
		Managers:   make(map[string]*Manager),

		globalPoolMu:          sync.RWMutex{},
		globalPool:            make(map[int32]chan []byte),
		globalPoolAccumulator: atomic.NewInt32(0),

		dedupeMu: sync.RWMutex{},
		Dedupe:   make(map[string]int64),

		IdentifyBuckets: bucketstore.NewBucketStore(),

		EventsInflight: atomic.NewInt32(0),

		State: NewSandwichState(),

		webhookBuckets: bucketstore.NewBucketStore(),
		statusCache:    interfacecache.NewInterfaceCache(),

		payloadPool: sync.Pool{
			New: func() interface{} { return new(sandwich_structs.SandwichPayload) },
		},

		receivedPool: sync.Pool{
			New: func() interface{} { return new(discord.GatewayPayload) },
		},

		sentPool: sync.Pool{
			New: func() interface{} { return new(discord.SentPayload) },
		},
	}

	sg.ctx, sg.cancel = context.WithCancel(context.Background())

	sg.Lock()
	defer sg.Unlock()

	configuration, err := sg.LoadConfiguration(sg.ConfigurationLocation)
	if err != nil {
		return nil, err
	}

	sg.configurationMu.Lock()
	defer sg.configurationMu.Unlock()

	sg.Configuration = configuration

	sg.Client = NewClient(baseURL, "")

	return sg, nil
}

// LoadConfiguration handles loading the configuration file.
func (sg *Sandwich) LoadConfiguration(path string) (configuration *SandwichConfiguration, err error) {
	sg.Logger.Debug().
		Str("path", path).
		Msg("Loading configuration")

	defer func() {
		if err == nil {
			sg.Logger.Info().Msg("Configuration loaded")
		}
	}()

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return configuration, ErrReadConfigurationFailure
	}

	configuration = &SandwichConfiguration{}

	err = yaml.Unmarshal(file, configuration)
	if err != nil {
		return configuration, ErrLoadConfigurationFailure
	}

	return configuration, nil
}

// SaveConfiguration handles saving the configuration file.
func (sg *Sandwich) SaveConfiguration(configuration *SandwichConfiguration, path string) (err error) {
	sg.Logger.Debug().Msg("Saving configuration")

	defer func() {
		if err == nil {
			sg.Logger.Info().Msg("Flushed configuration to disk")
		}
	}()

	data, err := yaml.Marshal(configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	err = ioutil.WriteFile(path, data, PermissionWrite)
	if err != nil {
		return fmt.Errorf("failed to write configuration to file: %w", err)
	}

	_ = sg.PublishGlobalEvent(sandwich_structs.SandwichEventConfigurationReload, nil)

	return nil
}

// Open starts up any listeners, configures services and starts up managers.
func (sg *Sandwich) Open() (err error) {
	sg.StartTime = time.Now().UTC()
	sg.Logger.Info().Msgf("Starting sandwich. Version %s", VERSION)

	go sg.PublishSimpleWebhook("Starting sandwich", "", "Version "+VERSION, EmbedColourSandwich)

	// Setup GRPC
	go sg.setupGRPC()

	// Setup Prometheus
	go sg.setupPrometheus()

	// Setup HTTP
	go sg.setupHTTP()

	sg.Logger.Info().Msg("Creating managers")
	sg.startManagers()

	return
}

// PublishGlobalEvent publishes an event to all Consumers.
func (sg *Sandwich) PublishGlobalEvent(eventType string, data jsoniter.RawMessage) (err error) {
	sg.globalPoolMu.RLock()
	defer sg.globalPoolMu.RUnlock()

	packet, _ := sg.payloadPool.Get().(*sandwich_structs.SandwichPayload)
	defer sg.payloadPool.Put(packet)

	packet.Op = discord.GatewayOpDispatch
	packet.Type = eventType
	packet.Data = data
	packet.Extra = make(map[string]jsoniter.RawMessage)

	packet.Metadata = sandwich_structs.SandwichMetadata{
		Version: VERSION,
	}

	payload, err := jsoniter.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to marshal packet: %w", err)
	}

	for _, pool := range sg.globalPool {
		pool <- payload
	}

	return
}

// Close closes all managers gracefully.
func (sg *Sandwich) Close() (err error) {
	sg.Logger.Info().Msg("Closing sandwich")

	go sg.PublishSimpleWebhook("Sandwich closing", "", "", EmbedColourSandwich)

	sg.managersMu.RLock()
	for _, manager := range sg.Managers {
		manager.Close()
	}
	sg.managersMu.RUnlock()

	if sg.cancel != nil {
		sg.cancel()
	}

	return nil
}

func (sg *Sandwich) startManagers() {
	sg.managersMu.Lock()

	for _, managerConfiguration := range sg.Configuration.Managers {
		if managerConfiguration.Identifier == "" {
			sg.Logger.Warn().Msg("Manager does not have an identifier. Ignoring")

			continue
		}

		if _, duplicate := sg.Managers[managerConfiguration.Identifier]; duplicate {
			sg.Logger.Warn().
				Str("identifier", managerConfiguration.Identifier).
				Msg("Manager contains duplicate identifier. Ignoring")

			go sg.PublishSimpleWebhook(
				"Manager contains duplicate identifier. Ignoring.",
				managerConfiguration.Identifier,
				"",
				EmbedColourWarning,
			)
		}

		manager := sg.NewManager(managerConfiguration)

		sg.Managers[managerConfiguration.Identifier] = manager

		err := manager.Initialize()
		if err != nil {
			manager.Error.Store(err.Error())

			manager.Logger.Error().Err(err).Msg("Failed to initialize manager")

			go sg.PublishSimpleWebhook("Failed to open manager", "`"+err.Error()+"`", "", EmbedColourDanger)
		} else if managerConfiguration.AutoStart {
			go manager.Open()
		}
	}

	sg.managersMu.Unlock()
}

func (sg *Sandwich) setupGRPC() (err error) {
	network := sg.Options.GRPCNetwork
	host := sg.Options.GRPCHost
	certpath := sg.Options.GRPCCertFile
	servernameoverride := sg.Options.GRPCServerNameOverride

	var grpcOptions []grpc.ServerOption

	if certpath != "" {
		var creds credentials.TransportCredentials

		creds, err = credentials.NewClientTLSFromFile(certpath, servernameoverride)
		if err != nil {
			sg.Logger.Error().Err(err).Msg("Failed to create new client TLS from file for gRPC")

			return
		}

		grpcOptions = append(grpcOptions, grpc.Creds(creds))
	}

	grpcListener := grpc.NewServer(grpcOptions...)
	grpcServer.RegisterSandwichServer(grpcListener, sg.newSandwichServer())
	reflection.Register(grpcListener)

	listener, err := net.Listen(network, host)
	if err != nil {
		sg.Logger.Error().Str("host", host).Err(err).Msg("Failed to bind to host")

		return
	}

	sg.Logger.Info().Msgf("Serving gRPC at %s", host)

	err = grpcListener.Serve(listener)
	if err != nil {
		sg.Logger.Error().Str("host", host).Err(err).Msg("Failed to serve gRPC server")

		return fmt.Errorf("failed to server grpc: %w", err)
	}

	return nil
}

func (sg *Sandwich) setupPrometheus() (err error) {
	prometheus.MustRegister(sandwichEventCount)
	prometheus.MustRegister(sandwichEventInflightCount)
	prometheus.MustRegister(sandwichEventBufferCount)
	prometheus.MustRegister(sandwichDispatchEventCount)
	prometheus.MustRegister(sandwichGatewayLatency)
	prometheus.MustRegister(sandwichUnavailableGuildCount)
	prometheus.MustRegister(sandwichStateTotalCount)
	prometheus.MustRegister(sandwichStateGuildCount)
	prometheus.MustRegister(sandwichStateGuildMembersCount)
	prometheus.MustRegister(sandwichStateRoleCount)
	prometheus.MustRegister(sandwichStateEmojiCount)
	prometheus.MustRegister(sandwichStateUserCount)
	prometheus.MustRegister(sandwichStateChannelCount)
	prometheus.MustRegister(grpcCacheRequests)
	prometheus.MustRegister(grpcCacheHits)
	prometheus.MustRegister(grpcCacheMisses)

	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{},
	))

	go sg.prometheusGatherer()
	go sg.cacheEjector()

	sg.Logger.Info().Msgf("Serving prometheus at %s", sg.Options.PrometheusAddress)

	err = http.ListenAndServe(sg.Options.PrometheusAddress, nil)
	if err != nil {
		sg.Logger.Error().Str("host", sg.Options.PrometheusAddress).Err(err).Msg("Failed to serve prometheus server")

		return fmt.Errorf("failed to serve prometheus: %w", err)
	}

	return nil
}

func (sg *Sandwich) setupHTTP() (err error) {
	sg.Logger.Info().Msgf("Serving http at %s", sg.Options.HTTPHost)

	cfg := session.NewDefaultConfig()
	provider, err := memory.New(memory.Config{})

	sg.SessionProvider = session.New(cfg)
	if err = sg.SessionProvider.SetProvider(provider); err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to set session provider")

		return fmt.Errorf("failed to set session provider: %w", err)
	}

	sg.RouterHandler, sg.DistHandler = sg.NewRestRouter()

	err = fasthttp.ListenAndServe(sg.Options.HTTPHost, sg.HandleRequest)
	if err != nil {
		sg.Logger.Error().Str("host", sg.Options.HTTPHost).Err(err).Msg("Failed to server http server")

		return fmt.Errorf("failed to serve webserver: %w", err)
	}

	return nil
}

func (sg *Sandwich) cacheEjector() {
	t := time.NewTicker(cacheEjectorInterval)

	for {
		select {
		case <-t.C:
			now := time.Now().Unix()

			// Guild Ejector
			allGuildIDs := make(map[discord.Snowflake]bool)

			sg.managersMu.RLock()
			for _, mg := range sg.Managers {
				mg.shardGroupsMu.RLock()
				for _, sg := range mg.ShardGroups {
					sg.guildsMu.RLock()
					for guildID, ok := range sg.Guilds {
						if ok {
							allGuildIDs[guildID] = true
						}
					}
					sg.guildsMu.RUnlock()
				}
				mg.shardGroupsMu.RUnlock()
			}
			sg.managersMu.RUnlock()

			ejectedGuilds := make([]discord.Snowflake, 0)

			sg.State.guildsMu.RLock()
			for guildID := range sg.State.Guilds {
				if val, ok := allGuildIDs[guildID]; !val || !ok {
					ejectedGuilds = append(ejectedGuilds, guildID)
				}
			}
			sg.State.guildsMu.RUnlock()

			ctx := &StateCtx{
				Stateless: true,
			}

			for _, guildID := range ejectedGuilds {
				sg.State.RemoveGuild(ctx, guildID)
			}

			ejectedDedupes := make([]string, 0)

			// MemberDedup Ejector
			sg.dedupeMu.RLock()
			for i, t := range sg.Dedupe {
				if now > t {
					ejectedDedupes = append(ejectedDedupes, i)
				}
			}
			sg.dedupeMu.RUnlock()

			if len(ejectedDedupes) > 0 {
				sg.dedupeMu.Lock()
				for _, i := range ejectedDedupes {
					delete(sg.Dedupe, i)
				}
				sg.dedupeMu.Unlock()
			}

			sg.Logger.Debug().
				Int("guildsEjected", len(ejectedGuilds)).
				Int("guildsTotal", len(allGuildIDs)).
				Int("ejectedDedupes", len(ejectedDedupes)).
				Msg("Ejected cache")
		}
	}
}

func (sg *Sandwich) prometheusGatherer() {
	t := time.NewTicker(prometheusGatherInterval)

	for {
		select {
		case <-t.C:
			sg.State.guildsMu.RLock()
			stateGuilds := len(sg.State.Guilds)
			sg.State.guildsMu.RUnlock()

			stateMembers := 0
			stateRoles := 0
			stateEmojis := 0
			stateChannels := 0

			sg.State.guildMembersMu.RLock()
			for _, guildMembers := range sg.State.GuildMembers {
				guildMembers.MembersMu.RLock()
				stateMembers += len(guildMembers.Members)
				guildMembers.MembersMu.RUnlock()
			}
			sg.State.guildMembersMu.RUnlock()

			sg.State.guildRolesMu.RLock()
			for _, guildRoles := range sg.State.GuildRoles {
				guildRoles.RolesMu.RLock()
				stateRoles += len(guildRoles.Roles)
				guildRoles.RolesMu.RUnlock()
			}
			sg.State.guildRolesMu.RUnlock()

			sg.State.guildEmojisMu.RLock()
			for _, guildEmojis := range sg.State.GuildEmojis {
				guildEmojis.EmojisMu.RLock()
				stateEmojis += len(guildEmojis.Emojis)
				guildEmojis.EmojisMu.RUnlock()
			}
			sg.State.guildEmojisMu.RUnlock()

			sg.State.guildChannelsMu.RLock()
			for _, guildChannels := range sg.State.GuildChannels {
				guildChannels.ChannelsMu.RLock()
				stateChannels += len(guildChannels.Channels)
				guildChannels.ChannelsMu.RUnlock()
			}
			sg.State.guildChannelsMu.RUnlock()

			sg.State.usersMu.RLock()
			stateUsers := len(sg.State.Users)
			sg.State.usersMu.RUnlock()

			sandwichStateTotalCount.Set(float64(
				stateGuilds + stateMembers + stateRoles + stateEmojis + stateUsers + stateChannels,
			))

			eventsInflight := sg.EventsInflight.Load()

			eventsBuffer := 0

			sg.managersMu.RLock()
			for _, manager := range sg.Managers {
				manager.shardGroupsMu.RLock()
				for _, shardgroup := range manager.ShardGroups {
					shardgroup.shardsMu.RLock()
					for _, shard := range shardgroup.Shards {
						eventsBuffer += len(shard.MessageCh)
					}
					shardgroup.shardsMu.RUnlock()
				}
				manager.shardGroupsMu.RUnlock()
			}
			sg.managersMu.RUnlock()

			sandwichStateGuildCount.Set(float64(stateGuilds))
			sandwichStateGuildMembersCount.Set(float64(stateMembers))
			sandwichStateRoleCount.Set(float64(stateRoles))
			sandwichStateEmojiCount.Set(float64(stateEmojis))
			sandwichStateUserCount.Set(float64(stateUsers))
			sandwichStateChannelCount.Set(float64(stateChannels))

			sandwichEventInflightCount.Set(float64(eventsInflight))
			sandwichEventBufferCount.Set(float64(eventsBuffer))

			sg.Logger.Debug().
				Int("guilds", stateGuilds).
				Int("members", stateMembers).
				Int("roles", stateRoles).
				Int("emojis", stateEmojis).
				Int("users", stateUsers).
				Int("channels", stateChannels).
				Int32("eventsInflight", eventsInflight).
				Int("eventsBuffer", eventsBuffer).
				Msg("Updated prometheus guages")
		}
	}
}
