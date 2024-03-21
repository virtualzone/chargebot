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

func sanityCheck() {
	log.Println("Running sanity check...")

	dbRefreshToken := GetDB().GetSetting(SettingRefreshToken)
	if dbRefreshToken == "" && GetConfig().TeslaRefreshToken == "" {
		log.Panicln("No Tesla Refresh Token found in database, initialize by setting env TESLA_REFRESH_TOKEN")
	}

	if dbRefreshToken == "" {
		GetDB().SetSetting(SettingRefreshToken, GetConfig().TeslaRefreshToken)
		log.Println("TESLA_REFRESH_TOKEN copied to database")
	}

	if GetConfig().CryptKey != "" && len(GetConfig().CryptKey) != 32 {
		log.Panicln("CRYPT_KEY must be 32 bytes long")
	}

	if GetConfig().Token == "" {
		log.Panicln("TOKEN not specified, get yours at https://chargebot.io")
	}

	if GetConfig().TokenPassword == "" {
		log.Panicln("PASSWORD for TOKEN not specified, get yours at https://chargebot.io")
	}

	if err := PingCommandServer(); err != nil {
		log.Panicf("Could not ping command server: %s\n", err.Error())
	}

	log.Println("Sanity check completed.")
}

func main() {
	log.Println("Starting chargebot.io worker node...")
	GetConfig().ReadConfig()
	GetDB().Connect()
	GetDB().InitDBStructure()
	sanityCheck()

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
