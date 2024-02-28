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
	GetOIDCProvider().Init()

	TeslaAPIInstance = &TeslaAPIImpl{}
	TeslaAPIInstance.InitTokenCache()

	NewChargeController().Init()
	InitPeriodicPriceUpdateControl()
	InitHTTPRouter()
	ServeHTTP()
	os.Exit(0)
}
