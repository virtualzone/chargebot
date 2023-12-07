package main

import (
	"log"
	"os"
)

func main() {
	log.Println("Starting Tesla Green Charge...")
	GetConfig().ReadConfig()
	ConnectDB()
	InitDBStructure()
	TeslaAPIInitTokenCache()
	InitPeriodicChargeControl()
	InitPeriodicPriceUpdateControl()
	ServeHTTP()
	os.Exit(0)
}
