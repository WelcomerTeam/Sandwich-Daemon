package main

import (
	"os"
	"os/signal"
	"syscall"

	gateway "github.com/TheRockettek/Sandwich-Daemon/internal"
)

func main() {
	sg, err := gateway.NewSandwich()
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
