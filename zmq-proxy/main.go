package main

import "log"

func main() {
	log.Println("Starting chargebot.io ZMQ Proxy...")
	GetConfig().ReadConfig()
	ServeZMQ()
}
