package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	internal "github.com/WelcomerTeam/Sandwich-Daemon/internal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"

	_ "github.com/joho/godotenv/autoload"
)

const (
	PermissionsDefault = 0o744

	int64Base    = 10
	int64BitSize = 64
)

func main() {
	prometheusAddress := flag.String("prometheusAddress", os.Getenv("PROMETHEUS_ADDRESS"), "Prometheus address")
	configurationPath := flag.String("configurationPath", os.Getenv("CONFIGURATION_PATH"), "Path of configuration file (default: sandwich.yaml)")
	gatewayURL := flag.String("gatewayURL", os.Getenv("GATEWAY_URL"), "Websocket for discord gateway")
	baseURL := flag.String("baseURL", os.Getenv("BASE_URL"), "BaseURL to send HTTP requests to. If empty, will use https://discord.com")

	grpcNetwork := flag.String("grpcNetwork", os.Getenv("GRPC_NETWORK"), "GRPC network type. The network must be \"tcp\", \"tcp4\", \"tcp6\", \"unix\" or \"unixpacket\".")
	grpcHost := flag.String("grpcHost", os.Getenv("GRPC_HOST"), "Host for GRPC.")
	grpcCertFile := flag.String("grpcCertFile", os.Getenv("GRPC_CERT_FILE"), "Optional cert file to use.")
	grpcServerNameOverride := flag.String("grpcServerNameOverride", os.Getenv("GRPC_SERVER_NAME_OVERRIDE"), "For testing only. If set to a non empty string, it will override the virtual host name of authority (e.g. :authority header field) in requests.")

	httpHost := flag.String("httpHost", os.Getenv("HTTP_HOST"), "Host to use for internal dashboard.")
	httpEnabled := flag.Bool("httpEnabled", MustParseBool(os.Getenv("HTTP_ENABLED")), "Enables the internal dashboard.")

	loggingLevel := flag.String("level", os.Getenv("LOGGING_LEVEL"), "Logging level")

	loggingFileLoggingEnabled := flag.Bool("fileLoggingEnabled", MustParseBool(os.Getenv("LOGGING_FILE_LOGGING_ENABLED")), "When enabled, will save logs to files")
	loggingEncodeAsJSON := flag.Bool("encodeAsJSON", MustParseBool(os.Getenv("LOGGING_ENCODE_AS_JSON")), "When enabled, will save logs as JSON")
	loggingCompress := flag.Bool("compress", MustParseBool(os.Getenv("LOGGING_COMPRESS")), "If true, will compress log files once reached max size")
	loggingDirectory := flag.String("directory", os.Getenv("LOGGING_DIRECTORY"), "Directory to store logs in")
	loggingFilename := flag.String("filename", os.Getenv("LOGGING_FILENAME"), "Filename to store logs as")
	loggingMaxSize := flag.Int("maxSize", MustParseInt(os.Getenv("LOGGING_MAX_SIZE")), "Maximum size for log files before being split into seperate files")
	loggingMaxBackups := flag.Int("maxBackups", MustParseInt(os.Getenv("LOGGING_MAX_BACKUPS")), "Maximum number of log files before being deleted")
	loggingMaxAge := flag.Int("maxAge", MustParseInt(os.Getenv("LOGGING_MAX_AGE")), "Maximum age in days for a log file")

	flag.Parse()

	// Setup Logger
	level, err := zerolog.ParseLevel(*loggingLevel)
	if err != nil {
		panic(fmt.Errorf(`failed to parse loggingLevel. zerolog.ParseLevel(%s): %w`, *loggingLevel, err))
	}

	zerolog.SetGlobalLevel(level)

	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.Stamp,
	}

	var writers []io.Writer

	writers = append(writers, writer)

	if *loggingFileLoggingEnabled {
		if err := os.MkdirAll(*loggingDirectory, PermissionsDefault); err != nil {
			log.Error().Err(err).Str("path", *loggingDirectory).Msg("Unable to create log directory")
		} else {
			lumber := &lumberjack.Logger{
				Filename:   path.Join(*loggingDirectory, *loggingFilename),
				MaxBackups: *loggingMaxBackups,
				MaxSize:    *loggingMaxSize,
				MaxAge:     *loggingMaxAge,
				Compress:   *loggingCompress,
			}

			if *loggingEncodeAsJSON {
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
	logger := zerolog.New(mw).With().Timestamp().Logger()
	logger.Info().Msg("Logging configured")

	options := internal.SandwichOptions{
		ConfigurationLocation: *configurationPath,
		PrometheusAddress:     *prometheusAddress,

		GRPCNetwork:            *grpcNetwork,
		GRPCHost:               *grpcHost,
		GRPCCertFile:           *grpcCertFile,
		GRPCServerNameOverride: *grpcServerNameOverride,

		HTTPHost:    *httpHost,
		HTTPEnabled: *httpEnabled,
	}

	if confGatewayURL, err := url.Parse(*gatewayURL); err == nil {
		options.GatewayURL = *confGatewayURL
	} else {
		panic(fmt.Sprintf(`url.Parse(%s): %v`, *baseURL, err))
	}

	if confBaseURL, err := url.Parse(*baseURL); err == nil {
		options.BaseURL = *confBaseURL
	} else {
		panic(fmt.Sprintf(`url.Parse(%s): %v`, *baseURL, err))
	}

	// Sandwich initialization
	sandwich, err := internal.NewSandwich(writer, options)
	if err != nil {
		logger.Panic().Err(err).Msg("Cannot create sandwich")
	}

	sandwich.Open()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-signalCh

	err = sandwich.Close()
	if err != nil {
		logger.Warn().Err(err).Msg("Exception whilst closing sandwich")
	}
}

func MustParseBool(str string) bool {
	boolean, _ := strconv.ParseBool(str)

	return boolean
}

func MustParseInt(str string) int {
	integer, _ := strconv.ParseInt(str, int64Base, int64BitSize)

	return int(integer)
}
