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
	/*
		GetTeslaAPI().CreateTelemetryConfig(GetDB().GetVehicleByVIN("LRWYGCEKXNC461719"))
			time.Sleep(5 * time.Second)
			GetTeslaAPI().GetTelemetryConfig(GetDB().GetVehicleByVIN("LRWYGCEKXNC461719"))
	*/
	ServeRPC()
	ServeHTTP()
	os.Exit(0)
}
