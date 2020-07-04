package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	gateway "github.com/TheRockettek/Sandwich-Daemon/internal"
	"github.com/rs/zerolog"
)

func main() {
	// zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.Stamp,
	}

	sg, err := gateway.NewSandwich(logger)
	if err != nil {
		println(err.Error())
	}
	err = sg.Open()
	if err != nil {
		println(err.Error())
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	err = sg.Close()
	if err != nil {
		println("Well, closing just errored but were literally about to close but heres the error:", err)
	}
}
