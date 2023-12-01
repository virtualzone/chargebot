package main

import (
	"log"
	"time"
)

var Ticker *time.Ticker = nil

func InitPeriodicChargeControl() {
	Ticker = time.NewTicker(time.Second * 10)
	go func() {
		for {
			<-Ticker.C
			PeriodicChargeControl()
		}
	}()
}

func PeriodicChargeControl() {
	log.Println("control...")
	vehicles := GetAllVehicles()
	for _, vehicle := range vehicles {
		state := GetVehicleState(vehicle.ID)
		log.Println(state)
	}
}
