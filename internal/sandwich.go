package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/WelcomerTeam/RealRock/bucketstore"
	"github.com/WelcomerTeam/RealRock/interfacecache"
	limiter "github.com/WelcomerTeam/RealRock/limiter"
	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/internal/structs"
	"github.com/fasthttp/session/v2"
	memory "github.com/fasthttp/session/v2/providers/memory"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
	"go.uber.org/atomic"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

// VERSION follows semantic versioning.
const VERSION = "1.17.0-antiraid"

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

var rawQuery = "v=10&encoding=json"
var gatewayURL = url.URL{
	Scheme: "wss",
	Host:   "gateway.discord.gg",
}

type Sandwich struct {
	Logger zerolog.Logger `json:"-"`

	// ReceivedPayload pool
	receivedPool sync.Pool
	// SentPayload pool
	sentPool sync.Pool

	gatewayLimiter limiter.DurationLimiter

	StartTime time.Time `json:"start_time" yaml:"start_time"`

	ctx    context.Context
	cancel func()

	ProducerClient *MQClient `json:"-"`

	IdentifyBuckets *bucketstore.BucketStore `json:"-"`

	EventsInflight *atomic.Int32 `json:"-"`

	Managers *csmap.CsMap[string, *Manager] `json:"managers" yaml:"managers"`

	Dedupe *csmap.CsMap[string, int64]

	State  *SandwichState `json:"-"`
	Client *Client        `json:"-"`

	webhookBuckets *bucketstore.BucketStore
	statusCache    *interfacecache.InterfaceCache

	SessionProvider *session.Session

	RouterHandler fasthttp.RequestHandler `json:"-"`
	DistHandler   fasthttp.RequestHandler `json:"-"`

	guildChunks Cache[discord.GuildID, GuildChunks]

	ConfigurationLocation string `json:"configuration_location"`

	Options SandwichOptions `json:"options" yaml:"options"`

	Configuration SandwichConfiguration `json:"configuration" yaml:"configuration"`

	configurationMu sync.RWMutex
	sync.Mutex
}

// SandwichConfiguration represents the configuration file.
type SandwichConfiguration struct {
	Identify struct {
		Headers map[string]string `json:"headers" yaml:"headers"`
		// URL allows for variables:
		// {shard_id}, {shard_count}, {token} {token_hash}, {max_concurrency}
		URL string `json:"url" yaml:"url"`
	} `json:"identify" yaml:"identify"`

	Producer struct {
		Configuration map[string]interface{} `json:"configuration" yaml:"configuration"`
		Type          string                 `json:"type" yaml:"type"`
	} `json:"producer" yaml:"producer"`

	HTTP struct {
		// OAuth config used to identification.
		OAuth *oauth2.Config `json:"oauth" yaml:"oauth"`

		// List of discord user IDs that can access the dashboard.
		UserAccess []string `json:"user_access" yaml:"user_access"`
	} `json:"http" yaml:"http"`

	Webhooks []string `json:"webhooks" yaml:"webhooks"`

	Managers []ManagerConfiguration `json:"managers" yaml:"managers"`
}

// SandwichOptions represents any options passable when creating the sandwich service.
type SandwichOptions struct {
	ConfigurationLocation string  `json:"configuration_location" yaml:"configuration_location"`
	PrometheusAddress     string  `json:"prometheus_address" yaml:"prometheus_address"`
	GatewayURL            url.URL `json:"gateway_url" yaml:"gateway_url"`

	// BaseURL to send HTTP requests to. If empty, will use https://discord.com
	BaseURL url.URL `json:"base_url" yaml:"base_url"`

	HTTPHost    string `json:"http_host" yaml:"http_host"`
	HTTPEnabled bool   `json:"http_enabled" yaml:"http_enabled"`
}

type GuildChunks struct {
	StartedAt   atomic.Time
	CompletedAt atomic.Time

	// Channel for receiving when chunks have been received.
	ChunkingChannel chan *discord.GuildMembersChunk

	// Only used for partials, stores number of chunks recieved
	ChunkCount atomic.Int32

	// Indicates if all chunks have been received.
	Complete atomic.Bool
}

type GuildChunkPartial struct {
	Nonce      string
	ChunkIndex int32
	ChunkCount int32
}

// NewSandwich creates the application state and initializes it.
func NewSandwich(logger io.Writer, options SandwichOptions) (sg *Sandwich, err error) {
	sg = &Sandwich{
		Logger: zerolog.New(logger).With().Timestamp().Logger(),

		ConfigurationLocation: options.ConfigurationLocation,

		configurationMu: sync.RWMutex{},
		Configuration:   SandwichConfiguration{},

		Options: options,

		gatewayLimiter: *limiter.NewDurationLimiter(1, time.Second),

		Managers: csmap.Create(
			csmap.WithSize[string, *Manager](10),
		),

		Dedupe: csmap.Create(
			csmap.WithSize[string, int64](10),
		),

		IdentifyBuckets: bucketstore.NewBucketStore(),

		EventsInflight: atomic.NewInt32(0),

		State: NewSandwichState(),

		webhookBuckets: bucketstore.NewBucketStore(),
		statusCache:    interfacecache.NewInterfaceCache(),

		receivedPool: sync.Pool{
			New: func() interface{} { return new(discord.GatewayPayload) },
		},

		sentPool: sync.Pool{
			New: func() interface{} { return new(discord.SentPayload) },
		},

		guildChunks: NewCache[discord.GuildID, GuildChunks](50),
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
func (sg *Sandwich) LoadConfiguration(path string) (configuration SandwichConfiguration, err error) {
	sg.Logger.Debug().
		Str("path", path).
		Msg("Loading configuration")

	defer func() {
		if err == nil {
			sg.Logger.Info().Msg("Configuration loaded")
		}
	}()

	file, err := os.ReadFile(path)
	if err != nil {
		return configuration, ErrReadConfigurationFailure
	}

	err = yaml.Unmarshal(file, &configuration)
	if err != nil {
		return configuration, ErrLoadConfigurationFailure
	}

	for _, mc := range configuration.Managers {
		if mc.Identifier == "" {
			return configuration, fmt.Errorf("manager does not have an identifier: %w", ErrLoadConfigurationFailure)
		}

		if mc.VirtualShards.Enabled && mc.VirtualShards.Count == 0 {
			return configuration, fmt.Errorf("manager %s has virtual shards enabled but shard count is 0: %w", mc.Identifier, ErrLoadConfigurationFailure)
		}
	}

	return configuration, nil
}

// SaveConfiguration handles saving the configuration file.
func (sg *Sandwich) SaveConfiguration(configuration *SandwichConfiguration, path string) error {
	sg.Logger.Debug().Msg("Saving configuration")

	data, err := yaml.Marshal(configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	err = os.WriteFile(path, data, PermissionWrite)
	if err != nil {
		return fmt.Errorf("failed to write configuration to file: %w", err)
	}

	_ = sg.PublishGlobalEvent(sandwich_structs.SandwichEventConfigurationReload, nil)

	return nil
}

// Open starts up any listeners, configures services and starts up managers.
func (sg *Sandwich) Open() {
	sg.StartTime = time.Now().UTC()
	sg.Logger.Info().Msgf("Starting sandwich. Version %s", VERSION)

	go sg.PublishSimpleWebhook("Starting sandwich", "", "Version "+VERSION, EmbedColourSandwich)

	// Setup Prometheus
	go sg.setupPrometheus()

	// Setup HTTP
	go sg.setupHTTP()

	sg.Logger.Info().Msg("Creating managers")
	sg.startManagers()
}

// PublishGlobalEvent publishes an event to all Consumers.
func (sg *Sandwich) PublishGlobalEvent(eventType string, data json.RawMessage) error {
	packet := &sandwich_structs.SandwichPayload{
		Metadata: &sandwich_structs.SandwichMetadata{
			Version: VERSION,
		},
		EventDispatchIdentifier: &sandwich_structs.EventDispatchIdentifier{
			GloballyRouted: true,
		},
		Op:   discord.GatewayOpDispatch,
		Type: eventType,
		Data: data,
	}

	sg.Managers.Range(func(key string, manager *Manager) bool {
		err := manager.RoutePayloadToConsumer(packet)

		if err != nil {
			sg.Logger.Error().Err(err).Msg("Failed to route payload to consumer")
		}

		err = manager.ProducerClient.Publish(sg.ctx, packet, manager.Configuration.Messaging.ChannelName)

		if err != nil {
			sg.Logger.Error().Err(err).Msg("Failed to publish event")
		}

		return false
	})

	return nil
}

// Close closes all managers gracefully.
func (sg *Sandwich) Close() error {
	sg.Logger.Info().Msg("Closing sandwich")

	go sg.PublishSimpleWebhook("Sandwich closing", "", "", EmbedColourSandwich)

	sg.Managers.Range(func(key string, manager *Manager) bool {
		manager.Close()
		return false
	})

	if sg.cancel != nil {
		sg.cancel()
	}

	return nil
}

func (sg *Sandwich) startManagers() {
	for _, managerConfiguration := range sg.Configuration.Managers {
		if managerConfiguration.Identifier == "" {
			sg.Logger.Warn().Msg("Manager does not have an identifier. Ignoring")

			continue
		}

		if _, duplicate := sg.Managers.Load(managerConfiguration.Identifier); duplicate {
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

		manager := sg.NewManager(&managerConfiguration)

		sg.Managers.Store(managerConfiguration.Identifier, manager)

		err := manager.Initialize(false)
		if err != nil {
			manager.Error.Store(err.Error())

			manager.Logger.Error().Err(err).Msg("Failed to initialize manager")

			go sg.PublishSimpleWebhook("Failed to open manager", "`"+err.Error()+"`", "", EmbedColourDanger)

			time.Sleep(5 * time.Second)

			// Retry
			sg.PublishSimpleWebhook("Retrying to open manager", "", "", EmbedColourWarning)
			sg.startManagers() // Recursively retry
		} else if managerConfiguration.AutoStart {
			go manager.Open()
		}
	}
}

func (sg *Sandwich) setupPrometheus() error {
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
	prometheus.MustRegister(sandwichStateVoiceStatesCount)
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

	err := http.ListenAndServe(sg.Options.PrometheusAddress, nil)
	if err != nil {
		sg.Logger.Error().Str("host", sg.Options.PrometheusAddress).Err(err).Msg("Failed to serve prometheus server")

		return fmt.Errorf("failed to serve prometheus: %w", err)
	}

	return nil
}

func (sg *Sandwich) setupHTTP() error {
	sg.Logger.Info().Msgf("Serving http at %s", sg.Options.HTTPHost)

	cfg := session.NewDefaultConfig()

	provider, err := memory.New(memory.Config{})
	if err != nil {
		return fmt.Errorf("failed to create memory provider: %w", err)
	}

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

var cacheEjectorStateCtx = StateCtx{
	Stateless: true,
}

func (sg *Sandwich) cacheEjector() {
	t := time.NewTicker(cacheEjectorInterval)

	for range t.C {
		now := time.Now().Unix()

		// Guild Ejector
		allGuildIDs := make(map[discord.GuildID]bool)

		sg.Managers.Range(func(key string, mg *Manager) bool {
			mg.ShardGroups.Range(func(i int32, sg *ShardGroup) bool {
				sg.Guilds.Range(func(guildID discord.GuildID, _ struct{}) bool {
					allGuildIDs[guildID] = true
					return false
				})
				return false
			})

			return false
		})

		ejectedGuilds := make([]discord.GuildID, 0)

		sg.State.Guilds.Range(func(guildID discord.GuildID, guild discord.Guild) bool {
			if val, ok := allGuildIDs[guildID]; !val || !ok {
				ejectedGuilds = append(ejectedGuilds, guildID)
			}
			return false
		})

		for _, guildID := range ejectedGuilds {
			sg.State.RemoveGuild(cacheEjectorStateCtx, guildID)
		}

		ejectedDedupes := make([]string, 0)

		// MemberDedup Ejector
		sg.Dedupe.Range(func(i string, t int64) bool {
			if now > t {
				ejectedDedupes = append(ejectedDedupes, i)
			}
			return false
		})

		if len(ejectedDedupes) > 0 {
			for _, i := range ejectedDedupes {
				sg.Dedupe.Delete(i)
			}
		}

		sg.Logger.Debug().
			Int("guildsEjected", len(ejectedGuilds)).
			Int("guildsTotal", len(allGuildIDs)).
			Int("ejectedDedupes", len(ejectedDedupes)).
			Msg("Ejected cache")
	}
}

func (sg *Sandwich) prometheusGatherer() {
	t := time.NewTicker(prometheusGatherInterval)

	for range t.C {
		stateGuilds := sg.State.Guilds.Count()

		stateMembers := 0
		stateRoles := 0
		stateEmojis := 0
		stateChannels := 0
		stateUsers := 0
		stateVoiceStates := 0

		sg.State.GuildMembers.Range(func(guildID discord.GuildID, guildMembers Cache[discord.UserID, discord.GuildMember]) bool {
			stateMembers += guildMembers.Count()
			return false
		})

		sg.State.GuildRoles.Range(func(guildID discord.GuildID, guildRoles Cache[discord.RoleID, discord.Role]) bool {
			stateRoles += guildRoles.Count()
			return false
		})

		sg.State.GuildEmojis.Range(func(guildID discord.GuildID, guildEmojis []discord.Emoji) bool {
			stateEmojis += len(guildEmojis)
			return false
		})

		sg.State.GuildChannels.Range(func(guildID discord.GuildID, guildChannels Cache[discord.ChannelID, discord.Channel]) bool {
			stateChannels += guildChannels.Count()
			return false
		})

		stateUsers = sg.State.Users.Count()

		sg.State.GuildVoiceStates.Range(func(guildID discord.GuildID, guildVoiceStates Cache[discord.UserID, discord.VoiceState]) bool {
			stateVoiceStates += guildVoiceStates.Count()
			return false
		})

		sandwichStateTotalCount.Set(float64(
			stateGuilds + stateMembers + stateRoles + stateEmojis + stateUsers + stateChannels + stateVoiceStates,
		))

		eventsInflight := sg.EventsInflight.Load()

		sandwichStateGuildCount.Set(float64(stateGuilds))
		sandwichStateGuildMembersCount.Set(float64(stateMembers))
		sandwichStateRoleCount.Set(float64(stateRoles))
		sandwichStateEmojiCount.Set(float64(stateEmojis))
		sandwichStateUserCount.Set(float64(stateUsers))
		sandwichStateChannelCount.Set(float64(stateChannels))
		sandwichStateVoiceStatesCount.Set(float64(stateVoiceStates))

		sandwichEventInflightCount.Set(float64(eventsInflight))

		sg.Logger.Debug().
			Int("guilds", stateGuilds).
			Int("members", stateMembers).
			Int("roles", stateRoles).
			Int("emojis", stateEmojis).
			Int("users", stateUsers).
			Int("channels", stateChannels).
			Int("voiceStates", stateVoiceStates).
			Int32("eventsInflight", eventsInflight).
			Msg("Updated prometheus gauges")
	}
}
