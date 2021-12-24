package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	internal "github.com/WelcomerTeam/Sandwich-Daemon/next/internal"
	"github.com/rs/zerolog"
)

func main() {
	lFlag := flag.String(
		"level", "info",
		"Global log level to use (debug/info/warn/error/fatal/panic/no/disabled/trace) (default: info)",
	)

	lConfigurationLocation := flag.String(
		"configuration", "sandwich.yaml",
		"Path of configuration file (default: sandwich.yaml)",
	)

	flag.Parse()

	// Parse flag and configure logging
	level, err := zerolog.ParseLevel(*lFlag)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.Stamp,
	}

	log := zerolog.New(consoleWriter).With().Timestamp().Logger()

	// Sandwich initialization
	sandwich, err := internal.NewSandwich(consoleWriter, *lConfigurationLocation)
	if err != nil {
		log.Panic().Err(err).Msg("Cannot create sandwich")
	}

	err = sandwich.Open()
	if err != nil {
		log.Panic().Err(err).Msg("Cannot open sandwich")
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-signalCh

	err = sandwich.Close()
	if err != nil {
		sandwich.Logger.Warn().Err(err).Msg("Exception whilst closing sandwich")
	}
}
