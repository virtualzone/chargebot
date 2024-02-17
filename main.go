package main

import (
	"log"
	"os"
)

var TeslaAPIInstance TeslaAPI

func GetTeslaAPI() TeslaAPI {
	return TeslaAPIInstance
}

func main() {
	log.Println("Starting chargebot.io backend...")
	GetConfig().ReadConfig()
	GetDB().Connect()
	if GetConfig().Reset {
		GetDB().ResetDBStructure()
	}
	GetDB().InitDBStructure()

	TeslaAPIInstance = &TeslaAPIImpl{}
	TeslaAPIInstance.InitTokenCache()

	NewChargeController().Init()
	InitPeriodicPriceUpdateControl()
	ServeZMQ()
	InitHTTPRouter()
	ServeHTTP()
	os.Exit(0)
}
