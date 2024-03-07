package main

import (
	"log"
	"os"
	"os/signal"
)

func main() {
	log.Println("Starting chargebot.io worker node...")
	GetConfig().ReadConfig()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	poller := &TelemetryPoller{
		Interrupt: make(chan os.Signal),
	}
	log.Println("1")
	poller.Poll()
	log.Println("2")
	for {
		select {
		case <-interrupt:
			poller.Interrupt <- os.Interrupt
			log.Println("Shutting down...")
			os.Exit(0)
		}
	}
}
