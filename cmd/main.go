package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	gateway "github.com/WelcomerTeam/Sandwich-Daemon/next/v2/internal"
	"github.com/rs/zerolog"
)

func main() {
	lFlag := flag.String("level", "info", "Global log level to use (debug/info/warn/error/fatal/panic/no/disabled/trace) (default: info)")
	lConfigurationLocation := flag.String("configuration", "sandwich.yaml", "Path of configuration file (default: sandwich.yaml)")
	lPoolConcurrency := flag.Int64("concurrency", 512, "Total number of events that can be processed concurrently (default: 512)")

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
	sandwich, err := gateway.NewSandwich(consoleWriter, *lConfigurationLocation, *lPoolConcurrency)
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
		sg.Logger.Warn().Err(err).Msg("Exception whilst closing sandwich")
	}
}
