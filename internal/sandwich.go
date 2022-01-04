package internal

import (
	"context"
	"github.com/WelcomerTeam/RealRock/bucketstore"
	limiter "github.com/WelcomerTeam/RealRock/limiter"
	discord "github.com/WelcomerTeam/Sandwich-Daemon/discord/structs"
	grpcServer "github.com/WelcomerTeam/Sandwich-Daemon/protobuf"
	structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"github.com/fasthttp/session/v2"
	memory "github.com/fasthttp/session/v2/providers/memory"
	jsoniter "github.com/json-iterator/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"go.uber.org/atomic"
	"golang.org/x/oauth2"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"time"
)

// VERSION follows semantic versionining.
const VERSION = "1.0.5"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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
	Host:   "discord.com",
	Scheme: "https",
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

	gatewayLimiter limiter.DurationLimiter `json:"-"`

	ProducerClient *MQClient `json:"-"`

	IdentifyBuckets *bucketstore.BucketStore `json:"-"`

	EventsInflight *atomic.Int64 `json:"-"`

	managersMu sync.RWMutex
	Managers   map[string]*Manager `json:"managers" yaml:"managers"`

	globalPoolMu          sync.RWMutex
	globalPool            map[int64]chan []byte
	globalPoolAccumulator *atomic.Int64

	dedupeMu sync.RWMutex
	Dedupe   map[string]int64

	State  *SandwichState `json:"-"`
	Client *Client        `json:"-"`

	webhookBuckets *bucketstore.BucketStore
	statusCache    *interfaceCache

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
	Logging struct {
		Level              string `json:"level" yaml:"level"`
		FileLoggingEnabled bool   `json:"file_logging_enabled" yaml:"file_logging_enabled"`

		EncodeAsJSON bool `json:"encode_as_json" yaml:"encode_as_json"`

		Directory  string `json:"directory" yaml:"directory"`
		Filename   string `json:"filename" yaml:"filename"`
		MaxSize    int    `json:"max_size" yaml:"max_size"`
		MaxBackups int    `json:"max_backups" yaml:"max_backups"`
		MaxAge     int    `json:"max_age" yaml:"max_age"`
		Compress   bool   `json:"compress" yaml:"compress"`
	} `json:"logging" yaml:"logging"`

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

	Prometheus struct {
		Host string `json:"host" yaml:"host"`
	} `json:"prometheus" yaml:"prometheus"`

	GRPC struct {
		Network string `json:"network" yaml:"network"`
		Host    string `json:"host" yaml:"host"`
	} `json:"grpc" yaml:"grpc"`

	HTTP struct {
		Host string `json:"host" yaml:"host"`

		// If enabled, allows access to dashboard else will only show
		// index page.
		Enabled bool `json:"enabled" yaml:"enabled"`

		// OAuth config used to identification.
		OAuth *oauth2.Config `json:"oauth" yaml:"oauth"`

		// List of discord user IDs that can access the dashboard.
		UserAccess []string `json:"user_access" yaml:"user_access"`
	} `json:"http" yaml:"http"`

	// BaseURL to send HTTP requests to. If empty, will use https://discord.com
	BaseURL string `json:"base" yaml:"base"`

	Webhooks []string `json:"webhooks" yaml:"webhooks"`

	Managers []*ManagerConfiguration `json:"managers" yaml:"managers"`
}

// NewSandwich creates the application state and initializes it.
func NewSandwich(logger io.Writer, configurationLocation string) (sg *Sandwich, err error) {
	sg = &Sandwich{
		Logger: zerolog.New(logger).With().Timestamp().Logger(),

		ConfigurationLocation: configurationLocation,

		configurationMu: sync.RWMutex{},
		Configuration:   &SandwichConfiguration{},

		gatewayLimiter: *limiter.NewDurationLimiter(1, time.Second),

		managersMu: sync.RWMutex{},
		Managers:   make(map[string]*Manager),

		globalPoolMu:          sync.RWMutex{},
		globalPool:            make(map[int64]chan []byte),
		globalPoolAccumulator: atomic.NewInt64(0),

		dedupeMu: sync.RWMutex{},
		Dedupe:   make(map[string]int64),

		IdentifyBuckets: bucketstore.NewBucketStore(),

		EventsInflight: atomic.NewInt64(0),

		State: NewSandwichState(),

		webhookBuckets: bucketstore.NewBucketStore(),
		statusCache:    NewInterfaceCache(),

		payloadPool: sync.Pool{
			New: func() interface{} { return new(structs.SandwichPayload) },
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

	if confBaseURL, err := url.Parse(configuration.BaseURL); err == nil {
		baseURL = *confBaseURL
		sg.Logger.Info().Str("url", baseURL.String()).Msg("BaseURL changed")
	}

	sg.Client = NewClient(baseURL, "")

	zlLevel, err := zerolog.ParseLevel(sg.Configuration.Logging.Level)
	if err != nil {
		sg.Logger.Warn().Str("level", sg.Configuration.Logging.Level).Msg("Logging level providied is not valid")
	} else {
		sg.Logger.Info().Str("level", sg.Configuration.Logging.Level).Msg("Changed logging level")
		zerolog.SetGlobalLevel(zlLevel)
	}

	// Create file and console logging

	var writers []io.Writer

	writers = append(writers, logger)

	if sg.Configuration.Logging.FileLoggingEnabled {
		if err := os.MkdirAll(sg.Configuration.Logging.Directory, PermissionsDefault); err != nil {
			log.Error().Err(err).Str("path", sg.Configuration.Logging.Directory).Msg("Unable to create log directory")
		} else {
			lumber := &lumberjack.Logger{
				Filename:   path.Join(sg.Configuration.Logging.Directory, sg.Configuration.Logging.Filename),
				MaxBackups: sg.Configuration.Logging.MaxBackups,
				MaxSize:    sg.Configuration.Logging.MaxSize,
				MaxAge:     sg.Configuration.Logging.MaxAge,
				Compress:   sg.Configuration.Logging.Compress,
			}

			if sg.Configuration.Logging.EncodeAsJSON {
				writers = append(writers, lumber)
			} else {
				writers = append(writers, zerolog.ConsoleWriter{
					Out:        lumber,
					TimeFormat: time.Stamp,
					NoColor:    true,
				})
			}
		}
	}

	mw := io.MultiWriter(writers...)
	sg.Logger = zerolog.New(mw).With().Timestamp().Logger()
	sg.Logger.Info().Msg("Logging configured")

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

	err = sg.ValidateConfiguration(configuration)
	if err != nil {
		return configuration, err
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
		return xerrors.Errorf("Failed to marshal configuration: %v", err)
	}

	err = ioutil.WriteFile(path, data, PermissionWrite)
	if err != nil {
		return xerrors.Errorf("Failed to write configuration to file: %v", err)
	}

	sg.PublishGlobalEvent("SW_CONFIGURATION_RELOAD", nil)

	return nil
}

// ValidateConfiguration ensures certain values in the configuration are passed.
func (sg *Sandwich) ValidateConfiguration(configuration *SandwichConfiguration) (err error) {
	if configuration.Prometheus.Host == "" {
		return ErrConfigurationValidatePrometheus
	}

	if configuration.GRPC.Host == "" {
		return ErrConfigurationValidateGRPC
	}

	if configuration.HTTP.Host == "" {
		return ErrConfigurationValidateHTTP
	}

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
func (sg *Sandwich) PublishGlobalEvent(eventType string, data interface{}) (err error) {
	sg.globalPoolMu.RLock()
	defer sg.globalPoolMu.RUnlock()

	packet, _ := sg.payloadPool.Get().(*structs.SandwichPayload)
	defer sg.payloadPool.Put(packet)

	packet.Op = discord.GatewayOpDispatch
	packet.Type = eventType
	packet.Data = data
	packet.Extra = make(map[string]interface{})

	packet.Metadata = structs.SandwichMetadata{
		Version: VERSION,
	}

	payload, err := json.Marshal(packet)
	if err != nil {
		return xerrors.Errorf("Failed to marshal packet: %w", err)
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
	sg.configurationMu.RLock()
	network := sg.Configuration.GRPC.Network
	host := sg.Configuration.GRPC.Host
	sg.configurationMu.RUnlock()

	listener, err := net.Listen(network, host)
	if err != nil {
		sg.Logger.Error().Str("host", host).Err(err).Msg("Failed to bind to host")

		return
	}

	var grpcOptions []grpc.ServerOption
	grpcListener := grpc.NewServer(grpcOptions...)
	grpcServer.RegisterSandwichServer(grpcListener, sg.newSandwichServer())

	sg.Logger.Info().Msgf("Serving gRPC at %s", host)

	err = grpcListener.Serve(listener)
	if err != nil {
		sg.Logger.Error().Str("host", host).Err(err).Msg("Failed to serve gRPC server")

		return xerrors.Errorf("Failed to server grpc: %v", err)
	}

	return nil
}

func (sg *Sandwich) setupPrometheus() (err error) {
	sg.configurationMu.RLock()
	host := sg.Configuration.Prometheus.Host
	sg.configurationMu.RUnlock()

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

	sg.Logger.Info().Msgf("Serving prometheus at %s", host)

	err = http.ListenAndServe(host, nil)
	if err != nil {
		sg.Logger.Error().Str("host", host).Err(err).Msg("Failed to serve prometheus server")

		return xerrors.Errorf("Failed to serve prometheus: %v", err)
	}

	return nil
}

func (sg *Sandwich) setupHTTP() (err error) {
	sg.configurationMu.RLock()
	host := sg.Configuration.HTTP.Host
	sg.configurationMu.RUnlock()

	sg.Logger.Info().Msgf("Serving http at %s", host)

	cfg := session.NewDefaultConfig()
	provider, err := memory.New(memory.Config{})

	sg.SessionProvider = session.New(cfg)
	if err = sg.SessionProvider.SetProvider(provider); err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to set session provider")

		return xerrors.Errorf("Failed to set session provider: %v", err)
	}

	sg.RouterHandler, sg.DistHandler = sg.NewRestRouter()

	err = fasthttp.ListenAndServe(host, sg.HandleRequest)
	if err != nil {
		sg.Logger.Error().Str("host", host).Err(err).Msg("Failed to server http server")

		return xerrors.Errorf("Failed to serve webserver: %v", err)
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
				Int64("eventsInflight", eventsInflight).
				Int("eventsBuffer", eventsBuffer).
				Msg("Updated prometheus guages")
		}
	}
}

func createDedupeMemberAddKey(guildID discord.Snowflake, memberID discord.Snowflake) (key string) {
	return "MA:" + guildID.String() + ":" + memberID.String()
}

func createDedupeMemberRemoveKey(guildID discord.Snowflake, memberID discord.Snowflake) (key string) {
	return "MR:" + guildID.String() + ":" + memberID.String()
}

// AddMemberDedupe creates a new dedupe.
func (sg *Sandwich) AddDedupe(key string) {
	sg.dedupeMu.Lock()
	sg.Dedupe[key] = time.Now().Add(memberDedupeExpiration).Unix()
	sg.dedupeMu.Unlock()
}

// CheckMemberDedupe returns if a dedupe is set. If true, event should be ignored.
func (sg *Sandwich) CheckDedupe(key string) (shouldDedupe bool) {
	sg.dedupeMu.RLock()
	value := sg.Dedupe[key]
	sg.dedupeMu.RUnlock()

	return time.Now().Unix() < value && value != 0
}

// RemoveMemberDedupe removes a dedupe.
func (sg *Sandwich) RemoveDedupe(key string) {
	sg.dedupeMu.Lock()
	delete(sg.Dedupe, key)
	sg.dedupeMu.Unlock()
}
