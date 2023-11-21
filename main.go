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
	ServeHTTP()
	os.Exit(0)
}
