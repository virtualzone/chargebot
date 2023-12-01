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
	InitPeriodicChargeControl()
	ServeHTTP()
	os.Exit(0)
}
