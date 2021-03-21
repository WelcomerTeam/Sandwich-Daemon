package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof" // nolint
	"os"
	"os/signal"
	"syscall"
	"time"

	gateway "github.com/TheRockettek/Sandwich-Daemon/internal"
	"github.com/rs/zerolog"
)

func main() {
	lFlag := flag.String("level", "info", "Log level to use (debug/info/warn/error/fatal/panic/no/disabled/trace)")

	flag.Parse()

	level, err := zerolog.ParseLevel(*lFlag)
	if err != nil {
		level = zerolog.InfoLevel
	}

	if level <= zerolog.DebugLevel {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	logger := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.Stamp,
	}

	log := zerolog.New(logger).With().Timestamp().Logger()

	if level != zerolog.NoLevel {
		log.Info().Str("logLevel", level.String()).Msg("Using logging")
	}

	zerolog.SetGlobalLevel(level)

	sg, err := gateway.NewSandwich(logger)
	if err != nil {
		log.Panic().Err(err).Msgf("Cannot create sandwich: %s", err)
	}

	err = sg.Open()

	if err != nil {
		log.Panic().Err(err).Msgf("Cannot open sandwich: %s", err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	err = sg.Close()

	if err != nil {
		sg.Logger.Error().Err(err).Msg("Exception whilst closing sandwich")
	}
}
