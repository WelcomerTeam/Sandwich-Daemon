package gateway

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	bucketstore "github.com/TheRockettek/Sandwich-Daemon/pkg/bucketStore"
	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/xerrors"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"
)

// VERSION respects semantic versioning
const VERSION = "0.1"

// ErrOnConfigurationFailure will return errors when loading configuration.
// If this is false, these errors are supressed. There is no reason for this
// to be false.
const ErrOnConfigurationFailure = true

// ConfigurationPath is the path to the file the configration will be located
// at.
const ConfigurationPath = "sandwich.yaml"

// SandwichConfiguration represents the configuration of the program
type SandwichConfiguration struct {
	Logging struct {
		ConsoleLoggingEnabled bool `json:"consolelogging" yaml:"consolelogging"`
		FileLoggingEnabled    bool `json:"filelogging" yaml:"filelogging"`

		EncodeAsJSON bool `json:"encodeasjson" yaml:"encodeasjson"` // Make the framework log as json

		Directory  string `json:"directory" yaml:"directory"`   // Directory to log into
		Filename   string `json:"filename" yaml:"filename"`     // Name of logfile
		MaxSize    int    `json:"maxsize" yaml:"maxsize"`       /// Size in MB before a new file
		MaxBackups int    `json:"maxbackups" yaml:"maxbackups"` // Number of files to keep
		MaxAge     int    `json:"maxage" yaml:"maxage"`         // Number of days to keep a logfile
	} `json:"logging" yaml:"logging"`

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
		Enabled bool   `json:"enabled" yaml:"enabled"`
		Host    string `json:"host" yaml:"host"`
	}

	Managers []*ManagerConfiguration `json:"managers" yaml:"managers"`
}

// Sandwich represents the global application state
type Sandwich struct {
	Logger zerolog.Logger `json:"-"`

	Start time.Time `json:"uptime"`

	Configuration *SandwichConfiguration `json:"configuration"`
	Managers      map[string]*Manager    `json:"managers"`

	// Buckets will be shared between all Managers
	Buckets *bucketstore.BucketStore `json:"-"`

	// Used for connection sharing
	RedisClient *redis.Client `json:"-"`
	NatsClient  *nats.Conn    `json:"-"`
	StanClient  stan.Conn     `json:"-"`
}

// NewSandwich creates the application state and initializes it
func NewSandwich(logger io.Writer) (sg *Sandwich, err error) {

	sg = &Sandwich{
		Logger:        zerolog.New(logger).With().Timestamp().Logger(),
		Configuration: &SandwichConfiguration{},
		Managers:      make(map[string]*Manager),
		Buckets:       bucketstore.NewBucketStore(),
	}

	err = sg.LoadConfiguration(ConfigurationPath)
	if err != nil {
		return nil, xerrors.Errorf("new sandwich: %w", err)
	}

	err = sg.NormalizeConfiguration()
	if err != nil {
		return nil, xerrors.Errorf("new sandwich: %w", err)
	}

	go sg.handleRequests()

	var writers []io.Writer

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

	mw := io.MultiWriter(writers...)
	sg.Logger = zerolog.New(mw).With().Timestamp().Logger()
	sg.Logger.Info().Msg("Logging configured")

	return
}

// LoadConfiguration loads the sandwich configuration
func (sg *Sandwich) LoadConfiguration(path string) (err error) {
	sg.Logger.Debug().Msg("Loading configuration")
	defer func() {
		if err == nil {
			sg.Logger.Info().Msg("Configuration loaded")
		}
	}()

	file, err := ioutil.ReadFile(path)
	if err != nil {
		if ErrOnConfigurationFailure {
			return xerrors.Errorf("load configuration readfile: %w", err)
		}
		sg.Logger.Warn().Msg("Failed to read configuration but ErrOnConfigurationFailure is disabled")
	}

	err = yaml.Unmarshal(file, sg.Configuration)
	if err != nil {
		if ErrOnConfigurationFailure {
			return xerrors.Errorf("load configuration unmarshal: %w", err)
		}
		sg.Logger.Warn().Msg("Failed to unmarshal configuration but ErrOnConfigurationFailure is disabled")
	}

	err = sg.NormalizeConfiguration()
	if err != nil {
		if ErrOnConfigurationFailure {
			return xerrors.Errorf("load configuration normalize: %w", err)
		}
		sg.Logger.Warn().Msg("Failed to normalize configuration but ErrOnConfigurationFailure is disabled")
	}

	return
}

// SaveConfiguration saves the sandwich configuration
func (sg *Sandwich) SaveConfiguration(path string) (err error) {
	sg.Logger.Debug().Msg("Saving configuration")
	defer func() {
		if err == nil {
			sg.Logger.Info().Msg("Flushed configuration to disk")
		}
	}()

	data, err := yaml.Marshal(sg.Configuration)
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
func (sg *Sandwich) NormalizeConfiguration() (err error) {
	// We will trim the password just incase
	sg.Configuration.Redis.Password = strings.TrimSpace(sg.Configuration.Redis.Password)

	if sg.Configuration.Redis.Address == "" {
		return xerrors.Errorf("Configurating missing Redis Address. Try 127.0.0.1:6379")
	}
	if sg.Configuration.NATS.Address == "" {
		return xerrors.New("Configuration missing NATS address. Try 127.0.0.1:4222")
	}
	if sg.Configuration.NATS.Channel == "" {
		return xerrors.Errorf("Configuration missing NATS channel. Try sandwich")
	}
	if sg.Configuration.NATS.Cluster == "" {
		return xerrors.Errorf("Configuration missing NATS cluster. Try cluster")
	}
	if sg.Configuration.HTTP.Host == "" {
		return xerrors.Errorf("Configuration missing HTTP host. Try 127.0.0.1:5469")
	}

	return
}

// Open starts up the application
func (sg *Sandwich) Open() (err error) {

	//          _-**--__
	//      _--*         *--__         Sandwich Daemon 0
	//  _-**                  **-_
	// |_*--_                _-* _|	   HTTP: 1
	// | *-_ *---_     _----* _-* |    Managers: 2
	//  *-_ *--__ *****  __---* _*
	//     *--__ *-----** ___--*       3
	//          **-____-**

	sg.Start = time.Now().UTC()
	sg.Logger.Info().Msgf("Starting sandwich\n\n         _-**--__\n     _--*         *--__         Sandwich Daemon %s\n _-**                  **-_\n|_*--_                _-* _|    HTTP: %s\n| *-_ *---_     _----* _-* |    Managers: %d\n *-_ *--__ *****  __---* _*\n     *--__ *-----** ___--*      %s\n         **-____-**\n",
		VERSION, sg.Configuration.HTTP.Host, len(sg.Configuration.Managers), "┬─┬ ノ( ゜-゜ノ)")

	// If we are not using unique clients, then we should create the shared connection now
	if !sg.Configuration.Redis.UniqueClients {
		sg.RedisClient = redis.NewClient(&redis.Options{
			Addr:     sg.Configuration.Redis.Address,
			Password: sg.Configuration.Redis.Password,
			DB:       sg.Configuration.Redis.DB,
		})
		sg.Logger.Info().Msg("Created standalone redis client")
	}

	for _, managerConfiguration := range sg.Configuration.Managers {
		// a, _ := json.Marshal(managerConfiguration)
		// println(string(a))

		manager, err := sg.NewManager(managerConfiguration)
		if err != nil {
			sg.Logger.Error().Err(err).Msg("Could not create manager")
			continue
		}
		if _, ok := sg.Managers[managerConfiguration.Identifier]; ok {
			sg.Logger.Warn().Str("identifer", managerConfiguration.Identifier).Msg("Found conflicting manager identifiers. Ignoring!")
			continue
		}

		sg.Managers[managerConfiguration.Identifier] = manager

		if manager.Configuration.AutoStart {
			go func() {
				err := manager.Open()
				if err != nil {
					manager.Logger.Error().Err(err).Msg("Failed to start up manager")
				}
			}()
		}
	}

	go func() {
		t := time.NewTicker(time.Second * 1)

		totalEvents := int64(0)

		// Store samples for 15 minute
		MaxSamples := 60 * 15
		samples := make([]int64, 0, MaxSamples)

		for {
			<-t.C

			eventCount := int64(0)
			managerCount := int64(0)
			for _, mg := range sg.Managers {
				managerCount = 0
				for _, sg := range mg.ShardGroups {
					for _, sh := range sg.Shards {
						managerCount += atomic.SwapInt64(sh.events, 0)
					}
				}
				eventCount += managerCount

				if mg.Analytics != nil {
					mg.Analytics.IncrementBy(managerCount)
					// We do not make an accumulator task so we manually ask to run the accumulator
					go mg.Analytics.RunOnce()
				}
			}
			totalEvents += eventCount

			samples = append(samples, eventCount)
			if len(samples) > MaxSamples {
				samples = samples[1:MaxSamples]
			}

			uptime := time.Now().UTC().Sub(sg.Start).Round(time.Second)

			// Get LastMinute Average
			// Get TotalAverage

			index := len(samples) - 60
			if index < 0 {
				index = 0
			}
			minuteEvents := int64(0)
			for _, val := range samples[index:] {
				minuteEvents += val
			}

			index = len(samples) - 900
			if index < 0 {
				index = 0
			}
			quarterEvents := int64(0)
			for _, val := range samples[index:] {
				quarterEvents += val
			}

			totalAverage := int64(float64(totalEvents) / uptime.Seconds())
			minuteAverage := int64(float64(minuteEvents) / 60)
			quarterAverage := int64(float64(quarterEvents) / 900)

			sg.Logger.Debug().Str("Elapsed", uptime.String()).Msgf("% 3d/s | AVG:% 3d | 1M:% 3d | 15M:% 3d", eventCount, totalAverage, minuteAverage, quarterAverage)

		}
	}()

	return
}

// Close will gracefully close the application
func (sg *Sandwich) Close() (err error) {
	sg.Logger.Info().Msg("Closing sandwich")

	// Close all managers
	for _, manager := range sg.Managers {
		manager.Close()
	}

	return
}
