package main

import (
	"log"
	"os"
)

func main() {
	log.Println("Starting Tesla Green Charge...")
	ServeHTTP()
	os.Exit(0)
}
