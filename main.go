package main

import (
	"log"
	"os"
)

func main() {
	log.Println("Starting Tesla Green Charge...")
	GetConfig().ReadConfig()
	ConnectDB()
	if GetConfig().Reset {
		ResetDBStructure()
	}
	InitDBStructure()
	TeslaAPIInitTokenCache()
	InitPeriodicChargeControl()
	InitPeriodicPriceUpdateControl()
	ServeHTTP()
	os.Exit(0)
}
