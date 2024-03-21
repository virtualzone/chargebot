package main

import (
	"log"
	"os"
	"os/signal"
)

var TeslaAPIInstance TeslaAPI

func GetTeslaAPI() TeslaAPI {
	return TeslaAPIInstance
}

func main() {
	log.Println("Starting chargebot.io worker node...")
	GetConfig().ReadConfig()
	GetDB().Connect()
	GetDB().InitDBStructure()

	TeslaAPIInstance = &TeslaAPIProxy{}

	InitHTTPRouter()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	poller := &TelemetryPoller{
		Interrupt: make(chan os.Signal),
	}
	poller.Poll()
	ServeHTTP()

	for {
		select {
		case <-interrupt:
			poller.Interrupt <- os.Interrupt
			log.Println("Shutting down...")
			os.Exit(0)
		}
	}
}
