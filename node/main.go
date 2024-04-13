package main

import (
	"log"
	"os"
	"os/signal"
)

var TeslaAPIInstance TeslaAPI
var poller *TelemetryPoller
var ChargeControllerInstance *ChargeController

func GetTeslaAPI() TeslaAPI {
	return TeslaAPIInstance
}

func GetTelemetryPoller() *TelemetryPoller {
	return poller
}

func GetChargeController() *ChargeController {
	return ChargeControllerInstance
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
		log.Panicf("Could not ping command server: %s - check if TOKEN and PASSWORD are correct, get yours at https://chargebot.io\n", err.Error())
	}

	log.Println("Sanity check completed.")
}

func main() {
	log.Println("Starting chargebot.io worker node...")
	GetConfig().ReadConfig()
	GetDB().Connect()
	GetDB().InitDBStructure()
	if GetConfig().InitDBOnly {
		return
	}
	sanityCheck()

	TeslaAPIInstance = &TeslaAPIProxy{}

	ChargeControllerInstance = NewChargeController()
	GetChargeController().Init()

	InitPeriodicPriceUpdateControl()

	InitHTTPRouter()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	poller = &TelemetryPoller{
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
