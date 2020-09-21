package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	gateway "github.com/TheRockettek/Sandwich-Daemon/internal"
	"github.com/rs/zerolog"
)

func main() {
	var lFlag = flag.String("level", "info", "Log level to use (debug/info/warn/error/fatal/panic/no/disabled/trace)")
	flag.Parse()

	level, err := zerolog.ParseLevel(*lFlag)

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
		log.Panic().Err(err).Send()
	}

	err = sg.Open()
	if err != nil {
		log.Panic().Err(err).Send()
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	err = sg.Close()
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Exception whilst closing sandwich")
	}
}
