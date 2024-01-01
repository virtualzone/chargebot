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
	log.Println("Starting Tesla Green Charge...")
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
	ServeHTTP()
	os.Exit(0)
}
