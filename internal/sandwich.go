package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bucketstore "github.com/TheRockettek/Sandwich-Daemon/pkg/bucketstore"
	consolepump "github.com/TheRockettek/Sandwich-Daemon/pkg/consolepump"
	"github.com/TheRockettek/Sandwich-Daemon/pkg/limiter"
	methodrouter "github.com/TheRockettek/Sandwich-Daemon/pkg/methodrouter"
	gatewayServer "github.com/TheRockettek/Sandwich-Daemon/protobuf"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	discord "github.com/TheRockettek/Sandwich-Daemon/structs/discord"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tevino/abool"
	"github.com/valyala/fasthttp"
	"golang.org/x/oauth2"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"
)

// VERSION respects semantic versioning.
const VERSION = "0.8.0+202103211705"

const (
	// ConfigurationPath is the path to the file the configration will be located
	// at.
	ConfigurationPath = "sandwich.yaml"

	// ErrOnConfigurationFailure will return errors when loading configuration.
	// If this is false, these errors are suppressed. There is no reason for this
	// to be false.
	ErrOnConfigurationFailure = true

	// Interval between each analytic sample.
	Interval = time.Second * 15

	// Samples to hold. 5 seconds and 720 samples is 1 hour.
	Samples = 720

	// distCacheDuration is the amount of hours to cache dist files.
	// This is 720 (1 month) by default.
	distCacheDuration = 720

	// Limit of how many events sandwich daemon can process concurrently on all
	// managers.
	poolConcurrency = 512

	// Relative location of the distribution file for the website.
	webRootPath = "web/dist"
)

// SandwichConfiguration represents the configuration of the program.
type SandwichConfiguration struct {
	Logging struct {
		Level                 string `json:"level" yaml:"level"`
		ConsoleLoggingEnabled bool   `json:"console_logging" yaml:"console_logging"`
		FileLoggingEnabled    bool   `json:"file_logging" yaml:"file_logging"`

		EncodeAsJSON bool `json:"encode_as_json" yaml:"encode_as_json"` // Make the framework log as json

		Directory  string `json:"directory" yaml:"directory"`     // Directory to log into.
		Filename   string `json:"filename" yaml:"filename"`       // Name of logfile.
		MaxSize    int    `json:"max_size" yaml:"max_size"`       // Size in MB before a new file.
		MaxBackups int    `json:"max_backups" yaml:"max_backups"` // Number of files to keep.
		MaxAge     int    `json:"max_age" yaml:"max_age"`         // Number of days to keep a logfile.

		MinimalWebhooks bool `json:"minimal_webhooks" yaml:"minimal_webhooks"`
		// If enabled, webhooks for status changes will use one liners instead of an embed.
	} `json:"logging" yaml:"logging"`

	RestTunnel struct {
		Enabled bool   `json:"enabled" yaml:"enabled"`
		URL     string `json:"url" yaml:"url"`
	} `json:"resttunnel" yaml:"resttunnel"`

	Redis struct {
		Address  string `json:"address" yaml:"address"`
		Password string `json:"password" yaml:"password"`
		DB       int    `json:"database" yaml:"database"`
		// If enabled, each manager will create their own redis connection
		// rather than sharing one.
		UniqueClients bool `json:"unique_clients" yaml:"unique_clients"`
	} `json:"redis" yaml:"redis"`

	Producer struct {
		Type          string                 `json:"type" yaml:"type"`
		Configuration map[string]interface{} `json:"configuration" yaml:"configuration"`
	} `json:"producer" yaml:"producer"`

	// NATS struct {
	// 	Address string `json:"address" yaml:"address"`
	// 	Channel string `json:"channel" yaml:"channel"`
	// 	Cluster string `json:"cluster" yaml:"cluster"`
	// } `json:"nats" yaml:"nats"`

	HTTP struct {
		Host          string `json:"host" yaml:"host"`
		SessionSecret string `json:"secret" yaml:"secret"`
		Enabled       bool   `json:"enabled" yaml:"enabled"`
		Public        bool   `json:"public" yaml:"public"`
	} `json:"http" yaml:"http"`

	Webhooks      []string       `json:"webhooks" yaml:"webhooks"`
	ElevatedUsers []string       `json:"elevated_users" yaml:"elevated_users"`
	OAuth         *oauth2.Config `json:"oauth" yaml:"oauth"`

	Managers []*ManagerConfiguration `json:"managers" yaml:"managers"`
}

// Sandwich represents the global application state.
type Sandwich struct {
	sync.RWMutex // used to block important functions

	Logger zerolog.Logger `json:"-"`

	Start time.Time `json:"uptime"`

	ConfigurationMu sync.RWMutex           `json:"-"`
	Configuration   *SandwichConfiguration `json:"configuration"`

	RestTunnelReverse abool.AtomicBool `json:"-"`
	RestTunnelEnabled abool.AtomicBool `json:"-"`

	ManagersMu sync.RWMutex        `json:"-"`
	Managers   map[string]*Manager `json:"-"`

	TotalEvents *int64 `json:"-"`

	// Buckets will be shared between all Managers
	Buckets *bucketstore.BucketStore `json:"-"`

	// Used for connection sharing
	RedisClient    *redis.Client `json:"-"`
	ProducerClient *MQClient     `json:"-"`

	Router *methodrouter.MethodRouter `json:"-"`
	Store  *sessions.CookieStore      `json:"-"`

	distHandler fasthttp.RequestHandler
	fs          *fasthttp.FS

	ConsolePump *consolepump.ConsolePump `json:"-"`

	Pool        *limiter.ConcurrencyLimiter `json:"-"`
	PoolWaiting *int64                      `json:"-"`
}

// NewSandwich creates the application state and initializes it.
func NewSandwich(logger io.Writer) (sg *Sandwich, err error) {
	sg = &Sandwich{
		Logger:          zerolog.New(logger).With().Timestamp().Logger(),
		ConfigurationMu: sync.RWMutex{},
		Configuration:   &SandwichConfiguration{},
		ManagersMu:      sync.RWMutex{},
		Managers:        make(map[string]*Manager),
		TotalEvents:     new(int64),
		Buckets:         bucketstore.NewBucketStore(),
		Pool:            limiter.NewConcurrencyLimiter("eventPool", poolConcurrency),
		PoolWaiting:     new(int64),
	}

	sg.Lock()
	defer sg.Unlock()

	configuration, err := sg.LoadConfiguration(ConfigurationPath)
	if err != nil {
		return nil, xerrors.Errorf("new sandwich: %w", err)
	}

	sg.ConfigurationMu.Lock()
	defer sg.ConfigurationMu.Unlock()

	sg.Configuration = configuration

	var writers []io.Writer

	zlLevel, err := zerolog.ParseLevel(sg.Configuration.Logging.Level)
	if err != nil {
		sg.Logger.Warn().
			Str("lvl", sg.Configuration.Logging.Level).
			Msg("Current zerolog level provided is not valid")
	} else {
		sg.Logger.Info().
			Str("lvl", sg.Configuration.Logging.Level).
			Msg("Changed logging level")
		zerolog.SetGlobalLevel(zlLevel)
	}

	if sg.Configuration.Logging.ConsoleLoggingEnabled {
		writers = append(writers, logger)
	}

	if sg.Configuration.Logging.FileLoggingEnabled {
		if err := os.MkdirAll(sg.Configuration.Logging.Directory, 0o744); err != nil {
			log.Error().Err(err).Str("path", sg.Configuration.Logging.Directory).Msg("Unable to create log directory")
		} else {
			lumber := &lumberjack.Logger{
				Filename:   path.Join(sg.Configuration.Logging.Directory, sg.Configuration.Logging.Filename),
				MaxBackups: sg.Configuration.Logging.MaxBackups,
				MaxSize:    sg.Configuration.Logging.MaxSize,
				MaxAge:     sg.Configuration.Logging.MaxAge,
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

	// We will only enable the ConsolePump if HTTP has been enabled
	if sg.Configuration.HTTP.Enabled {
		sg.ConsolePump = consolepump.NewConsolePump()
		writers = append(writers, sg.ConsolePump)
	}

	mw := io.MultiWriter(writers...)
	sg.Logger = zerolog.New(mw).With().Timestamp().Logger()
	sg.Logger.Info().Msg("Logging configured")

	return sg, nil
}

// LoadConfiguration loads the sandwich configuration.
func (sg *Sandwich) LoadConfiguration(path string) (configuration *SandwichConfiguration, err error) {
	sg.Logger.Debug().Msg("Loading configuration")

	defer func() {
		if err == nil {
			sg.Logger.Info().Msg("Configuration loaded")
		}
	}()

	file, err := ioutil.ReadFile(path)
	if err != nil {
		if ErrOnConfigurationFailure {
			return configuration, xerrors.Errorf("load configuration readfile: %w", err)
		}

		sg.Logger.Warn().Msg("Failed to read configuration but ErrOnConfigurationFailure is disabled")
	}

	configuration = &SandwichConfiguration{}
	err = yaml.Unmarshal(file, &configuration)

	if err != nil {
		if ErrOnConfigurationFailure {
			return configuration, xerrors.Errorf("load configuration unmarshal: %w", err)
		}

		sg.Logger.Warn().Msg("Failed to unmarshal configuration but ErrOnConfigurationFailure is disabled")
	}

	err = sg.NormalizeConfiguration(configuration)

	if err != nil {
		if ErrOnConfigurationFailure {
			return configuration, xerrors.Errorf("load configuration normalize: %w", err)
		}

		sg.Logger.Warn().Msg("Failed to normalize configuration but ErrOnConfigurationFailure is disabled")
	}

	return configuration, err
}

// SaveConfiguration saves the sandwich configuration.
func (sg *Sandwich) SaveConfiguration(configuration *SandwichConfiguration, path string) (err error) {
	sg.Logger.Debug().Msg("Saving configuration")

	defer func() {
		if err == nil {
			sg.Logger.Info().Msg("Flushed configuration to disk")
		}
	}()

	// Load old configuration only if necessary
	var config *SandwichConfiguration

	oldmanagers := make(map[string]*ManagerConfiguration)

	for _, manager := range configuration.Managers {
		if !manager.Persist {
			config, err = sg.LoadConfiguration(path)

			if err != nil {
				return err
			}

			for _, mg := range config.Managers {
				oldmanagers[mg.Identifier] = mg
			}
		}
	}

	storedManagers := []*ManagerConfiguration{}

	for _, manager := range configuration.Managers {
		if !manager.Persist {
			oldmanager, ok := oldmanagers[manager.Identifier]
			// If we do not persist, reuse the old configuration. If it does not exist we do not need to store anything.
			if ok {
				manager = oldmanager
			} else {
				continue
			}
		}

		storedManagers = append(storedManagers, manager)
	}

	configuration.Managers = storedManagers

	data, err := yaml.Marshal(configuration)
	if err != nil {
		return xerrors.Errorf("save configuration marshal: %w", err)
	}

	err = ioutil.WriteFile(path, data, 0o600)

	if err != nil {
		return xerrors.Errorf("save configuration write: %w", err)
	}

	return nil
}

// NormalizeConfiguration fills in any defaults within the configuration.
func (sg *Sandwich) NormalizeConfiguration(configuration *SandwichConfiguration) (err error) {
	// We will trim the password just in case.
	// sg.ConfigurationMu.Lock()
	// defer sg.ConfigurationMu.Unlock()
	configuration.Redis.Password = strings.TrimSpace(configuration.Redis.Password)

	if configuration.Redis.Address == "" {
		return xerrors.Errorf("Configuration missing Redis Address. Try 127.0.0.1:6379")
	}

	// if configuration.NATS.Address == "" {
	// 	return xerrors.New("Configuration missing producer address. Try 127.0.0.1:4222")
	// }

	// if configuration.NATS.Channel == "" {
	// 	return xerrors.Errorf("Configuration missing producer channel. Try sandwich")
	// }

	// if configuration.NATS.Cluster == "" {
	// 	return xerrors.Errorf("Configuration missing producer cluster. Try cluster")
	// }

	if configuration.HTTP.Host == "" {
		return xerrors.Errorf("Configuration missing HTTP host. Try 127.0.0.1:5469")
	}

	return
}

// Open starts up sandwich and loads the configuration, starts up the HTTP server and
// managers who have autostart enabled.
func (sg *Sandwich) Open() (err error) {
	//          _-**--__
	//      _--*         *--__         Sandwich Daemon 0
	//  _-**                  **-_
	// |_*--_                _-* _|	   HTTP: 127.0.0.1:1234
	// | *-_ *---_     _----* _-* |    Managers: 2
	//  *-_ *--__ *****  __---* _*
	//     *--__ *-----** ___--*       placeholder
	//          **-____-**
	sg.ConfigurationMu.RLock()
	defer sg.ConfigurationMu.RUnlock()

	sg.Start = time.Now().UTC()
	sg.Logger.Info().Msgf("Starting sandwich\n\n         _-**--__\n"+
		"     _--*         *--__         Sandwich Daemon %s\n"+
		" _-**                  **-_\n"+
		"|_*--_                _-* _|    HTTP: %s\n"+
		"| *-_ *---_     _----* _-* |    Managers: %d\n"+
		" *-_ *--__ *****  __---* _*\n"+
		"     *--__ *-----** ___--*      %s\n"+
		"         **-____-**\n",
		VERSION, sg.Configuration.HTTP.Host, len(sg.Configuration.Managers), "┬─┬ ノ( ゜-゜ノ)")

	// Check if HTTP is enabled and if it is, set it up.
	if sg.Configuration.HTTP.Enabled {
		if sg.Configuration.HTTP.Public {
			sg.Logger.Warn().Msg(
				"Public mode is enabled on the HTTP API. This can allow anyone to" +
					"get bot credentials if exposed publicly. It is recommended you disable" +
					"this and add trusted user ids in sandwich.yaml under \"elevated_users\"")
		}

		sg.Logger.Info().Msg("Starting up http server")
		sg.fs = &fasthttp.FS{
			Root:            webRootPath,
			Compress:        true,
			CompressBrotli:  true,
			AcceptByteRange: true,
			CacheDuration:   time.Hour * distCacheDuration,
			PathNotFound:    fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {}),
		}
		sg.distHandler = sg.fs.NewRequestHandler()

		sg.Store = sessions.NewCookieStore([]byte(sg.Configuration.HTTP.SessionSecret))

		sg.Logger.Info().Msg("Creating endpoints")
		sg.Router = createEndpoints(sg)

		go func() {
			fmt.Printf("Serving dashboard on %s (Press CTRL+C to quit)\n", sg.Configuration.HTTP.Host)

			err = fasthttp.ListenAndServe(sg.Configuration.HTTP.Host, sg.HandleRequest)
			if err != nil {
				sg.Logger.Error().Str("host", sg.Configuration.HTTP.Host).Err(err).Msg("Failed to serve http server")
			}
		}()
	} else {
		sg.Logger.Info().Msg("The web interface will not start as HTTP is disabled in the configuration")
	}

	go func() {
		lis, err := net.Listen("tcp", "localhost:10000")
		if err != nil {
			sg.Logger.Error().Str("host", "localhost:10000").Err(err).Msg("Failed to bind address for gRPC server")

			return
		}

		var opts []grpc.ServerOption
		grpcServer := grpc.NewServer(opts...)
		gatewayServer.RegisterGatewayServer(grpcServer, sg.NewGatewayServer())

		fmt.Printf("Serving gRPC on %s (Press CTRL+C to quit)\n", "localhost:10000")

		err = grpcServer.Serve(lis)
		if err != nil {
			sg.Logger.Error().Str("host", "localhost:10000").Err(err).Msg("Failed to serve gRPC server")
		}
	}()

	sg.Logger.Info().Msg("Creating standalone redis client")
	sg.RedisClient = redis.NewClient(&redis.Options{
		Addr:     sg.Configuration.Redis.Address,
		Password: sg.Configuration.Redis.Password,
		DB:       sg.Configuration.Redis.DB,
	})

	// Configure RestTunnel
	sg.Logger.Info().Msg("Configuring RestTunnel")

	if sg.Configuration.RestTunnel.Enabled {
		enabled, reverse, err := sg.VerifyRestTunnel(sg.Configuration.RestTunnel.URL)
		if err != nil {
			sg.Logger.Error().Err(err).Msg("Failed to verify RestTunnel")
		}

		sg.RestTunnelReverse.SetTo(reverse)
		sg.RestTunnelEnabled.SetTo(enabled)
	} else {
		sg.RestTunnelEnabled.UnSet()
	}

	go sg.PublishWebhook(context.Background(), discord.WebhookMessage{
		Embeds: []discord.Embed{
			{
				Title:       "Starting up Sandwich-Daemon",
				URL:         "https://github.com/TheRockettek/Sandwich-Daemon",
				Description: fmt.Sprintf("Version %s", VERSION),
				Color:       discord.EmbedSandwich,
				Timestamp:   WebhookTime(time.Now().UTC()),
			},
		},
	})

	sg.Logger.Info().Msg("Creating managers")

	sg.startManagers()

	go sg.gatherAnalytics()
	go sg.analyticsRunner()

	return nil
}

func (sg *Sandwich) startManagers() {
	sg.ManagersMu.Lock()

	for _, managerConfiguration := range sg.Configuration.Managers {
		manager, err := sg.NewManager(managerConfiguration)
		if err != nil {
			sg.Logger.Error().Err(err).Msg("Could not create manager")

			continue
		}

		if _, ok := sg.Managers[managerConfiguration.Identifier]; ok {
			sg.Logger.Warn().
				Str("identifier", managerConfiguration.Identifier).
				Msg("Found conflicting manager identifiers. Ignoring!")

			go sg.PublishWebhook(context.Background(), discord.WebhookMessage{
				Embeds: []discord.Embed{
					{
						Title:     "Found conflicting manager identifiers. Ignoring!",
						Color:     discord.EmbedWarning,
						Timestamp: WebhookTime(time.Now().UTC()),
						Footer: &discord.EmbedFooter{
							Text: fmt.Sprintf("Manager %s",
								manager.Configuration.DisplayName),
						},
					},
				},
			})

			continue
		}

		sg.Managers[managerConfiguration.Identifier] = manager
		err = manager.Open()

		if err != nil {
			manager.ErrorMu.Lock()
			manager.Error = err.Error()
			manager.ErrorMu.Unlock()

			sg.Logger.Error().Err(err).Msg("Failed to start up manager")
		} else if manager.Configuration.AutoStart {
			go func() {
				manager.Gateway, err = manager.GetGateway()
				if err != nil {
					manager.Logger.Error().Err(err).Msg("Failed to start up manager")

					return
				}

				manager.GatewayMu.RLock()
				manager.Logger.Info().
					Int("sessions", manager.Gateway.SessionStartLimit.Remaining).
					Msg("Retrieved gateway information")

				shardCount := manager.GatherShardCount()
				if shardCount >= manager.Gateway.SessionStartLimit.Remaining {
					manager.Logger.Error().Err(ErrSessionLimitExhausted).Msg("Failed to start up manager")
					manager.GatewayMu.RUnlock()

					go sg.PublishWebhook(context.Background(), discord.WebhookMessage{
						Embeds: []discord.Embed{
							{
								Title: fmt.Sprintf("Failed to start up manager as not enough sessions to start %d shard(s). %d remain",
									shardCount, manager.Gateway.SessionStartLimit.Remaining),
								Color:     discord.EmbedDanger,
								Timestamp: WebhookTime(time.Now().UTC()),
								Footer: &discord.EmbedFooter{
									Text: fmt.Sprintf("Manager %s",
										manager.Configuration.DisplayName),
								},
							},
						},
					})

					return
				}

				manager.GatewayMu.RUnlock()

				ready, err := manager.Scale(manager.GenerateShardIDs(shardCount), shardCount, true)
				if err != nil {
					manager.Logger.Error().Err(err).Msg("Failed to start up manager")

					go sg.PublishWebhook(context.Background(), discord.WebhookMessage{
						Embeds: []discord.Embed{
							{
								Title:       "Failed to start up manager",
								Description: err.Error(),
								Color:       discord.EmbedDanger,
								Timestamp:   WebhookTime(time.Now().UTC()),
								Footer: &discord.EmbedFooter{
									Text: fmt.Sprintf("Manager %s",
										manager.Configuration.DisplayName),
								},
							},
						},
					})

					return
				}

				// Wait for all shards in ShardGroup to be ready
				<-ready
			}()
		}
	}

	sg.ManagersMu.Unlock()
}

func (sg *Sandwich) gatherAnalytics() {
	var events int64

	var managerEvents int64

	t := time.NewTicker(time.Second * 1)

	for {
		<-t.C

		events = 0

		sg.ManagersMu.RLock()

		for _, mg := range sg.Managers {
			managerEvents = 0

			mg.ShardGroupsMu.RLock()
			for _, sg := range mg.ShardGroups {
				sg.ShardsMu.RLock()
				for _, sh := range sg.Shards {
					managerEvents += atomic.SwapInt64(sh.events, 0)
				}
				sg.ShardsMu.RUnlock()
			}
			mg.ShardGroupsMu.RUnlock()

			mg.AnalyticsMu.RLock()
			if mg.Analytics != nil {
				mg.Analytics.IncrementBy(managerEvents)
			}
			mg.AnalyticsMu.RUnlock()

			events += managerEvents
		}
		sg.ManagersMu.RUnlock()
		atomic.AddInt64(sg.TotalEvents, events)

		now := time.Now().UTC()
		uptime := now.Sub(sg.Start).Round(time.Second)
		sg.Logger.Debug().Str("Elapsed", uptime.String()).Int64("q", atomic.LoadInt64(sg.PoolWaiting)).Msgf("%d/s", events)
	}
}

func (sg *Sandwich) analyticsRunner() {
	e := time.NewTicker(Interval)

	for {
		<-e.C

		now := time.Now().UTC().Round(time.Second)

		sg.ManagersMu.RLock()

		for _, mg := range sg.Managers {
			mg.AnalyticsMu.RLock()
			if mg.Analytics != nil {
				go mg.Analytics.RunOnce(now)
			}
			mg.AnalyticsMu.RUnlock()
		}
		sg.ManagersMu.RUnlock()
	}
}

// VerifyRestTunnel returns a boolean if RestTunnel can be found and is alive.
func (sg *Sandwich) VerifyRestTunnel(restTunnelURL string) (enabled bool, reverse bool, err error) {
	if restTunnelURL == "" {
		return
	}

	_url, err := url.Parse(restTunnelURL)
	if err != nil {
		sg.Logger.Error().Err(err).Msgf("Failed to parse RestTunnel URL %s", restTunnelURL)

		return
	}

	resp, err := http.Get(_url.Scheme + "://" + _url.Host + "/resttunnel") //nolint:noctx
	if err != nil {
		sg.Logger.Error().Err(err).Msgf("Failed to connect to RestTunnel")

		return
	}

	baseResponse := structs.RestTunnelAliveResponse{}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&baseResponse)

	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to parse RestTunnel alive response")
	}

	aliveResponse := baseResponse.Data

	sg.Logger.Info().Msgf(
		"\\o/ Detected RestTunnel version %s. Running in reverse mode: %t",
		aliveResponse.Version, aliveResponse.Reverse)

	return true, aliveResponse.Reverse, nil
}

// Close will gracefully close the application.
func (sg *Sandwich) Close() (err error) {
	sg.Logger.Info().Msg("Closing sandwich")

	go sg.PublishWebhook(context.Background(), discord.WebhookMessage{
		Embeds: []discord.Embed{
			{
				Title:     "Shutting down sandwich",
				Color:     discord.EmbedSandwich,
				Timestamp: WebhookTime(time.Now().UTC()),
			},
		},
	})

	// Close all managers
	sg.ManagersMu.RLock()
	for _, manager := range sg.Managers {
		manager.Close()
	}
	sg.ManagersMu.RUnlock()

	return
}

// PublishWebhook sends a webhook message to all added webhooks in the configuration.
func (sg *Sandwich) PublishWebhook(ctx context.Context, message discord.WebhookMessage) {
	for _, webhook := range sg.Configuration.Webhooks {
		_, err := sg.SendWebhook(ctx, webhook, message)
		if err != nil && !xerrors.Is(err, context.Canceled) {
			sg.Logger.Warn().Err(err).Str("url", webhook).Msg("Failed to send webhook")
		}
	}
}

// SendWebhook executes a webhook request. This does not currently support sending.
// files.
func (sg *Sandwich) SendWebhook(ctx context.Context, _url string,
	message discord.WebhookMessage) (status int, err error) {
	var c *Client

	// We will trim whitespace just in case.
	_url = strings.TrimSpace(_url)

	_, err = url.Parse(_url)
	if err != nil {
		return -1, xerrors.Errorf("failed to parse webhook URL: %w", err)
	}

	if sg.RestTunnelEnabled.IsSet() {
		c = NewClient("", sg.Configuration.RestTunnel.URL, sg.RestTunnelReverse.IsSet(), false)
	} else {
		c = NewClient("", "", false, false)
	}

	res, err := json.Marshal(message)
	if err != nil {
		return -1, xerrors.Errorf("failed to marshal webhook message: %w", err)
	}

	_, status, err = c.Fetch(ctx, "POST", _url, bytes.NewBuffer(res), map[string]string{
		"Content-Type": "application/json",
	})

	return status, err
}

// TestWebhook tests a URL to see if it is valid.
func (sg *Sandwich) TestWebhook(ctx context.Context,
	_url string) (status int, err error) {
	_, err = url.Parse(_url)
	if err != nil {
		return -1, xerrors.Errorf("failed to parse webhook URL: %w", err)
	}

	message := discord.WebhookMessage{
		Content: "Test Webhook",
		Embeds: []discord.Embed{
			{
				Title:       "This is a test webhook",
				Description: "This is the description",
				URL:         "https://github.com/TheRockettek/Sandwich-Daemon",
				Thumbnail: &discord.EmbedThumbnail{
					URL: "https://raw.githubusercontent.com/TheRockettek/Sandwich-Daemon/master/" +
						webRootPath + "/img/icons/favicon-96x96.png",
				},
				Color:     discord.EmbedSandwich,
				Timestamp: WebhookTime(time.Now().UTC()),
			},
		},
	}

	return sg.SendWebhook(ctx, _url, message)
}
