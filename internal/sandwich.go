package internal

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	limiter "github.com/WelcomerTeam/RealRock/limiter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tevino/abool"
	"go.uber.org/atomic"
	"golang.org/x/xerrors"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

const VERSION = "0.0.1"

type Sandwich struct {
	sync.RWMutex

	Logger    zerolog.Logger
	StartTime time.Time

	ConfigurationLocation atomic.String

	ConfigurationMu sync.RWMutex
	Configuration   *SandwichConfiguration

	// RestTunnel is a third party library that handles the ratelimiting.
	// RestTunnel can accept either a direct URL or path only (when running in reverse mode)
	// https://github.com/WelcomerTeam/RestTunnel
	RestTunnelEnabled  abool.AtomicBool
	RestTunnelOnlyPath abool.AtomicBool

	ProducerClient *MQClient

	// EventPool contains the global event pool limiter defined on startup flags.
	// EventPoolWaiting stores any events that are waiting for a spot.
	EventPool        *limiter.ConcurrencyLimiter
	EventPoolWaiting atomic.Int64
	EventPoolLimit   int

	ManagersMu sync.RWMutex
	Managers   map[string]*Manager

	State *SandwichState
}

// SandwichConfiguration represents the configuration file
type SandwichConfiguration struct {
	Logging struct {
		Level              string
		FileLoggingEnabled bool

		EncodeAsJSON bool

		Directory  string
		Filename   string
		MaxSize    int
		MaxBackups int
		MaxAge     int
		Compress   bool

		MinimalWebhooks bool
	}

	State struct {
		StoreGuildMembers bool
		StoreEmojis       bool

		EnableSmaz bool
	}

	Identify struct {
		// URL allows for variables:
		// {shard}, {shard_count}, {auth}, {manager_name}, {shard_group_id}
		URL string

		Headers map[string]string
	}

	RestTunnel struct {
		Enabled bool
		URL     string
	}

	Producer struct {
		Type          string
		Configuration map[string]interface{}
	}

	GRPC struct {
		Network string
		Host    string
	}

	Webhooks []string

	Managers []*ManagerConfiguration
}

// NewSandwich creates the application state and initializes it
func NewSandwich(logger io.Writer, configurationLocation string, eventPoolLimit int) (sg *Sandwich, err error) {
	sg = &Sandwich{
		Logger: zerolog.New(logger).With().Timestamp().Logger(),

		ConfigurationMu: sync.RWMutex{},
		Configuration:   &SandwichConfiguration{},

		ManagersMu: sync.RWMutex{},
		Managers:   make(map[string]*Manager),

		EventPool:        limiter.NewConcurrencyLimiter(eventPoolLimit),
		EventPoolWaiting: *atomic.NewInt64(0),
		EventPoolLimit:   eventPoolLimit,

		State: NewSandwichState(),
	}

	sg.Lock()
	defer sg.Unlock()

	configuration, err := sg.LoadConfiguration(configurationLocation)
	if err != nil {
		return nil, err
	}

	sg.ConfigurationMu.Lock()
	defer sg.ConfigurationMu.Unlock()

	sg.Configuration = configuration

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
		if err := os.MkdirAll(sg.Configuration.Logging.Directory, 0o744); err != nil {
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

// LoadConfiguration handles loading the configuration file
func (sg *Sandwich) LoadConfiguration(path string) (configuration *SandwichConfiguration, err error) {
	sg.Logger.Debug().Msg("Loading configuration")

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

	err = yaml.Unmarshal(file, &configuration)
	if err != nil {
		return configuration, ErrLoadConfigurationFailure
	}

	err = sg.ValidateConfiguration(configuration)
	if err != nil {
		return configuration, err
	}

	return configuration, nil
}

// SaveConfiguration handles saving the configuration file
func (sg *Sandwich) SaveConfiguration(configuration *SandwichConfiguration, path string) (err error) {
	sg.Logger.Debug().Msg("Saving configuration")

	defer func() {
		if err == nil {
			sg.Logger.Info().Msg("Flushed configuration to disk")
		}
	}()

	data, err := yaml.Marshal(configuration)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, data, 0o600)
	if err != nil {
		return err
	}

	return nil
}

// ValidateConfiguration ensures certain values in the configuration are passed
func (sg *Sandwich) ValidateConfiguration(configuration *SandwichConfiguration) (err error) {
	if configuration.Identify.URL == "" {
		return ErrConfigurationValidateIdentify
	}

	if configuration.RestTunnel.Enabled && configuration.RestTunnel.URL == "" {
		return ErrConfigurationValidateRestTunnel
	}

	if configuration.GRPC.Host == "" {
		return ErrConfigurationValidateGRPC
	}

	return nil
}

// Open starts up any listeners, configures services and starts up managers.
func (sg *Sandwich) Open() (err error) {
	sg.ConfigurationMu.RLock()
	defer sg.ConfigurationMu.RUnlock()

	sg.StartTime = time.Now().UTC()
	sg.Logger.Info().Msgf("Starting sandwich. Version %s", VERSION)

	// Setup GRPC
	err = sg.setupGRPC()
	if err != nil {
		return err
	}

	// Setup RestTunnel
	err = sg.setupRestTunnel()
	if err != nil {
		return err
	}

}

func (sg *Sandwich) setupGRPC() (err error) {
	// listener, err := net.Listen(sg.Configuration.GRPC.Network, sg.Configuration.GRPC.Host)
	// if err != nil {
	// 	sg.Logger.Error().Str("host", sg.Configuration.GRPC.Host).Err(err).Msg("Failed to bind to host")

	// 	return
	// }

	// var grpcOptions []grpc.ServerOptions
	// grpcListener := grpc.NewServer(opts...)
	// grpcServer.RegisterGatewayServer(grpcListener, sg.NewGatewayServer())

	// err = grpcListener.Serve(listener)
	// if err != nil {
	// 	sg.Logger.Error().Str("host", sg.Configuration.GRPC.Host).Err(err).Msg("Failed to serve gRPC server")

	// 	return
	// }

	// sg.Logger.Info().Msgf("Serving gRPC at %s", sg.Configuration.GRPC.Host)

	return nil
}

func (sg *Sandwich) setupRestTunnel() (err error) {
	if sg.Configuration.RestTunnel.Enabled {
		enabled, reverse, err := sg.VerifyRestTunnel(sg.Configuration.RestTunnel.URL)
		if err != nil {
			sg.Logger.Error().Err(err).Msg("Failed to verify RestTunnel")
		}

		sg.RestTunnelOnlyPath.SetTo(reverse)
		sg.RestTunnelEnabled.SetTo(enabled)
	} else {
		sg.RestTunnelEnabled.UnSet()
	}

	return nil
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

// SendWebhook executes a webhook request. Does not support sending files.
func (sg *Sandwich) SendWebhook(ctx context.Context, webhookUrl string, message discord.WebhookMessage) (status int, err error) {

}
