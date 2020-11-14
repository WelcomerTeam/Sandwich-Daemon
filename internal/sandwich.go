package gateway

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bucketstore "github.com/TheRockettek/Sandwich-Daemon/pkg/bucketStore"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/sessions"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tevino/abool"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"golang.org/x/oauth2"
	"golang.org/x/xerrors"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"
)

// VERSION respects semantic versioning
const VERSION = "0.2"

// ErrOnConfigurationFailure will return errors when loading configuration.
// If this is false, these errors are suppressed. There is no reason for this
// to be false.
const ErrOnConfigurationFailure = true

// ConfigurationPath is the path to the file the configration will be located
// at.
const ConfigurationPath = "sandwich.yaml"

// Interval between each analytic sample
const Interval = time.Second * 15

// Samples to hold. 5 seconds and 720 samples is 1 hour
const Samples = 720

// SandwichConfiguration represents the configuration of the program
type SandwichConfiguration struct {
	Logging struct {
		ConsoleLoggingEnabled bool `json:"console_logging" yaml:"console_logging"`
		FileLoggingEnabled    bool `json:"file_logging" yaml:"file_logging"`

		EncodeAsJSON bool `json:"encode_as_json" yaml:"encode_as_json"` // Make the framework log as json

		Directory  string `json:"directory" yaml:"directory"`     // Directory to log into
		Filename   string `json:"filename" yaml:"filename"`       // Name of logfile
		MaxSize    int    `json:"max_size" yaml:"max_size"`       /// Size in MB before a new file
		MaxBackups int    `json:"max_backups" yaml:"max_backups"` // Number of files to keep
		MaxAge     int    `json:"max_age" yaml:"max_age"`         // Number of days to keep a logfile
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

	NATS struct {
		Address string `json:"address" yaml:"address"`
		Channel string `json:"channel" yaml:"channel"`
		Cluster string `json:"cluster" yaml:"cluster"`
	} `json:"nats" yaml:"nats"`

	HTTP struct {
		Enabled       bool   `json:"enabled" yaml:"enabled"`
		Host          string `json:"host" yaml:"host"`
		SessionSecret string `json:"secret" yaml:"secret"`
		Public        bool   `json:"public" yaml:"public"`
	} `json:"http" yaml:"http"`

	ElevatedUsers []int64        `json:"elevated_users" yaml:"elevated_users"`
	OAuth         *oauth2.Config `json:"oauth" yaml:"oauth"`

	Managers []*ManagerConfiguration `json:"managers" yaml:"managers"`
}

// Sandwich represents the global application state
type Sandwich struct {
	Logger zerolog.Logger `json:"-"`

	Start time.Time `json:"uptime"`

	ConfigurationMu sync.RWMutex           `json:"-"`
	Configuration   *SandwichConfiguration `json:"configuration"`

	RestTunnelReverse      abool.AtomicBool `json:"-"`
	RestTunnelEnabled      abool.AtomicBool `json:"-"`
	RestTunnelEnabledValue bool             `json:"rest_tunnel_enabled"`

	ManagersMu sync.RWMutex        `json:"-"`
	Managers   map[string]*Manager `json:"managers"`

	TotalEvents *int64 `json:"-"`

	// Buckets will be shared between all Managers
	Buckets *bucketstore.BucketStore `json:"-"`

	// Used for connection sharing
	RedisClient *redis.Client `json:"-"`
	NatsClient  *nats.Conn    `json:"-"`
	StanClient  stan.Conn     `json:"-"`

	Router *MethodRouter         `json:"-"`
	Store  *sessions.CookieStore `json:"-"`

	distHandler fasthttp.RequestHandler
	fs          *fasthttp.FS
}

// NewSandwich creates the application state and initializes it
func NewSandwich(logger io.Writer) (sg *Sandwich, err error) {

	sg = &Sandwich{
		Logger:          zerolog.New(logger).With().Timestamp().Logger(),
		ConfigurationMu: sync.RWMutex{},
		Configuration:   &SandwichConfiguration{},
		ManagersMu:      sync.RWMutex{},
		Managers:        make(map[string]*Manager),
		TotalEvents:     new(int64),
		Buckets:         bucketstore.NewBucketStore(),
	}

	configuration, err := sg.LoadConfiguration(ConfigurationPath)
	if err != nil {
		return nil, xerrors.Errorf("new sandwich: %w", err)
	}
	sg.ConfigurationMu.Lock()
	sg.Configuration = configuration
	sg.ConfigurationMu.Unlock()

	var writers []io.Writer
	sg.ConfigurationMu.RLock()
	if sg.Configuration.Logging.ConsoleLoggingEnabled {
		writers = append(writers, logger)
	}
	if sg.Configuration.Logging.FileLoggingEnabled {
		if err := os.MkdirAll(sg.Configuration.Logging.Directory, 0744); err != nil {
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
	sg.ConfigurationMu.RUnlock()

	mw := io.MultiWriter(writers...)
	sg.Logger = zerolog.New(mw).With().Timestamp().Logger()
	sg.Logger.Info().Msg("Logging configured")

	return
}

// HandleRequest handles incoming HTTP requests
func (sg *Sandwich) HandleRequest(ctx *fasthttp.RequestCtx) {
	start := time.Now()
	var processingMS int64

	defer func() {
		var log *zerolog.Event
		statusCode := ctx.Response.StatusCode()
		switch {
		case (statusCode >= 400 && statusCode <= 499):
			log = sg.Logger.Warn()
		case (statusCode >= 500 && statusCode <= 599):
			log = sg.Logger.Error()
		default:
			log = sg.Logger.Info()
		}
		log.Msgf("%s %s %s %d %d %dms",
			ctx.RemoteAddr(),
			ctx.Request.Header.Method(),
			ctx.Request.URI().PathOriginal(),
			statusCode,
			len(ctx.Response.Body()),
			processingMS,
		)
	}()

	fasthttp.CompressHandlerBrotliLevel(func(ctx *fasthttp.RequestCtx) {
		fasthttpadaptor.NewFastHTTPHandler(sg.Router)(ctx)
		if ctx.Response.StatusCode() != 404 {
			ctx.SetContentType("application/json;charset=utf8")
		}
		// If there is no URL in router then try serving from the dist
		// folder.
		if ctx.Response.StatusCode() == 404 {
			ctx.Response.Reset()
			sg.distHandler(ctx)
		}
		// If there is no URL in router or in dist then send index.html
		if ctx.Response.StatusCode() == 404 {
			ctx.Response.Reset()
			ctx.SendFile("web/dist/index.html")
		}
	}, fasthttp.CompressBrotliBestCompression, fasthttp.CompressBestCompression)(ctx)

	processingMS = time.Now().Sub(start).Milliseconds()
	ctx.Response.Header.Set("X-Elapsed", strconv.FormatInt(processingMS, 10))
}

// LoadConfiguration loads the sandwich configuration
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

	return
}

// SaveConfiguration saves the sandwich configuration
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

	err = ioutil.WriteFile(path, data, 0644)
	if err != nil {
		return xerrors.Errorf("save configuration write: %w", err)
	}

	return
}

// NormalizeConfiguration fills in any defaults within the configuration
func (sg *Sandwich) NormalizeConfiguration(configuration *SandwichConfiguration) (err error) {
	// We will trim the password just incase
	sg.ConfigurationMu.Lock()
	configuration.Redis.Password = strings.TrimSpace(sg.Configuration.Redis.Password)
	sg.ConfigurationMu.Unlock()

	sg.ConfigurationMu.RLock()
	defer sg.ConfigurationMu.RUnlock()

	if configuration.Redis.Address == "" {
		return xerrors.Errorf("Configuration missing Redis Address. Try 127.0.0.1:6379")
	}
	if configuration.NATS.Address == "" {
		return xerrors.New("Configuration missing NATS address. Try 127.0.0.1:4222")
	}
	if configuration.NATS.Channel == "" {
		return xerrors.Errorf("Configuration missing NATS channel. Try sandwich")
	}
	if configuration.NATS.Cluster == "" {
		return xerrors.Errorf("Configuration missing NATS cluster. Try cluster")
	}
	if configuration.HTTP.Host == "" {
		return xerrors.Errorf("Configuration missing HTTP host. Try 127.0.0.1:5469")
	}

	return
}

// Open starts up the application
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
	sg.Logger.Info().Msgf("Starting sandwich\n\n         _-**--__\n     _--*         *--__         Sandwich Daemon %s\n _-**                  **-_\n|_*--_                _-* _|    HTTP: %s\n| *-_ *---_     _----* _-* |    Managers: %d\n *-_ *--__ *****  __---* _*\n     *--__ *-----** ___--*      %s\n         **-____-**\n",
		VERSION, sg.Configuration.HTTP.Host, len(sg.Configuration.Managers), "┬─┬ ノ( ゜-゜ノ)")

	if sg.Configuration.HTTP.Enabled {
		if sg.Configuration.HTTP.Public {
			sg.Logger.Warn().Msg("Public mode is enabled on the HTTP API. This can allow anyone to get bot credentials if exposed publicly. It is recommended you disable this and add trusted user ids in sandwich.yaml under \"elevated_users\"")
		}

		sg.Logger.Info().Msg("Starting up http server")
		sg.fs = &fasthttp.FS{
			Root:               "web/dist",
			IndexNames:         []string{"index.html"},
			GenerateIndexPages: true,
			Compress:           true,
			AcceptByteRange:    true,
			CacheDuration:      time.Hour * 24,
			PathNotFound:       fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) { return }),
		}
		sg.distHandler = sg.fs.NewRequestHandler()

		sg.Store = sessions.NewCookieStore([]byte(sg.Configuration.HTTP.SessionSecret))

		sg.Logger.Info().Msg("Creating endpoints")
		sg.Router = createEndpoints(sg)

		go func() {
			fmt.Printf("Serving on %s (Press CTRL+C to quit)\n", sg.Configuration.HTTP.Host)
			err = fasthttp.ListenAndServe(sg.Configuration.HTTP.Host, sg.HandleRequest)
			if err != nil {
				sg.Logger.Error().Err(err).Msg("Failed to start up http server")
			}
		}()
	} else {
		sg.Logger.Info().Msg("The web interface will not start as HTTP is disabled in the configuration")
	}

	sg.Logger.Info().Msg("Creating standalone redis client")
	sg.RedisClient = redis.NewClient(&redis.Options{
		Addr:     sg.Configuration.Redis.Address,
		Password: sg.Configuration.Redis.Password,
		DB:       sg.Configuration.Redis.DB,
	})

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

	sg.Logger.Info().Msg("Creating managers")
	for _, managerConfiguration := range sg.Configuration.Managers {

		manager, err := sg.NewManager(managerConfiguration)
		if err != nil {
			sg.Logger.Error().Err(err).Msg("Could not create manager")
			continue
		}
		sg.ManagersMu.RLock()
		if _, ok := sg.Managers[managerConfiguration.Identifier]; ok {
			sg.Logger.Warn().Str("identifer", managerConfiguration.Identifier).Msg("Found conflicting manager identifiers. Ignoring!")
			continue
		}
		sg.ManagersMu.RUnlock()

		sg.ManagersMu.Lock()
		sg.Managers[managerConfiguration.Identifier] = manager
		sg.ManagersMu.Unlock()

		err = manager.Open()
		if err != nil {
			manager.Error = err.Error()
			sg.Logger.Error().Err(err).Msg("Failed to start up manager")
		} else {
			if manager.Configuration.AutoStart {
				go func() {
					manager.Gateway, err = manager.GetGateway()
					if err != nil {
						manager.Logger.Error().Err(err).Msg("Failed to start up manager")
						return
					}

					manager.GatewayMu.RLock()
					manager.Logger.Info().Int("sessions", manager.Gateway.SessionStartLimit.Remaining).Msg("Retrieved gateway information")

					shardCount := manager.GatherShardCount()
					if shardCount >= manager.Gateway.SessionStartLimit.Remaining {
						manager.Logger.Error().Err(ErrSessionLimitExhausted).Msg("Failed to start up manager")
						manager.GatewayMu.RUnlock()
						return
					}
					manager.GatewayMu.RUnlock()

					ready, err := manager.Scale(manager.GenerateShardIDs(shardCount), shardCount, true)
					if err != nil {
						manager.Logger.Error().Err(err).Msg("Failed to start up manager")
						return
					}

					// Wait for all shards in ShardGroup to be ready
					<-ready
				}()
			}
		}
	}

	go func() {
		var events int64
		var managerEvents int64
		t := time.NewTicker(time.Second * 1)
		for {
			<-t.C
			events = 0
			sg.ManagersMu.RLock()
			for _, mg := range sg.Managers {
				managerEvents = 0

				mg.ShardGroupMu.Lock()
				for _, sg := range mg.ShardGroups {
					for _, sh := range sg.Shards {
						managerEvents += atomic.SwapInt64(sh.events, 0)
					}
				}
				mg.ShardGroupMu.Unlock()

				if mg.Analytics != nil {
					mg.Analytics.IncrementBy(managerEvents)
				}
				events += managerEvents
			}
			sg.ManagersMu.RUnlock()
			atomic.AddInt64(sg.TotalEvents, events)
			now := time.Now().UTC()
			uptime := now.Sub(sg.Start).Round(time.Second)
			sg.Logger.Debug().Str("Elapsed", uptime.String()).Msgf("%d/s", events)
		}
	}()

	go func() {
		e := time.NewTicker(Interval)
		for {
			<-e.C
			now := time.Now().UTC().Round(time.Second)
			sg.ManagersMu.RLock()
			for _, mg := range sg.Managers {
				if mg.Analytics != nil {
					go mg.Analytics.RunOnce(now)
				}
			}
			sg.ManagersMu.RUnlock()
		}
	}()

	return
}

// VerifyRestTunnel returns a boolean if RestTunnel can be found and is alive
func (sg *Sandwich) VerifyRestTunnel(restTunnelURL string) (enabled bool, reverse bool, err error) {
	if restTunnelURL == "" {
		return
	}

	_url, err := url.Parse(restTunnelURL)
	if err != nil {
		sg.Logger.Error().Err(err).Msgf("Failed to parse RestTunnel URL %s", restTunnelURL)
		return
	}

	resp, err := http.Get(_url.Scheme + "://" + _url.Host + "/resttunnel")
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
	sg.Logger.Info().Msgf("\\o/ Detected RestTunnel version %s. Running in reverse mode: %t", aliveResponse.Version, aliveResponse.Reverse)
	return true, aliveResponse.Reverse, nil
}

// Close will gracefully close the application
func (sg *Sandwich) Close() (err error) {
	sg.Logger.Info().Msg("Closing sandwich")

	// Close all managers
	sg.ManagersMu.RLock()
	for _, manager := range sg.Managers {
		manager.Close()
	}
	sg.ManagersMu.RUnlock()

	return
}
