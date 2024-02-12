package main

import (
	"log"
	"time"
)

var TickerPriceUpdate *time.Ticker = nil

func InitPeriodicPriceUpdateControl() {
	TickerPriceUpdate = time.NewTicker(time.Minute * 6)
	go func() {
		for {
			PeriodicPriceUpdateControl()
			<-TickerPriceUpdate.C
		}
	}()
}

func PeriodicPriceUpdateControl() {
	PeriodicPriceUpdateControl_Tibber()
}

func PeriodicPriceUpdateControl_Tibber() {
	log.Println("Updating Tibber prices...")

	// First, care about the vehicles that don't even have prices for today
	// Limit to 45 vehicles so we don't exceed the API limits
	l := GetDB().GetVehicleIDsWithTibberTokenWithoutPricesForToday(45)
	log.Println(l)
	for _, vehicleID := range l {
		vehicle := GetDB().GetVehicleByID(vehicleID)
		log.Printf("Updating today's Tibber prices for vehicle ID %d ...\n", vehicleID)
		PeriodicPriceUpdateControlProcessVehicle_Tibber(vehicle)
	}

	now := time.Now().UTC()
	if now.Hour() > 12 {
		// Next, if it's past 13:00 GMT, handle the vehicle's without prices for tomorrow
		l := GetDB().GetVehicleIDsWithTibberTokenWithoutPricesForTomorrow(45)
		log.Println(l)
		for _, vehicleID := range l {
			vehicle := GetDB().GetVehicleByID(vehicleID)
			log.Printf("Updating tomorrow's Tibber prices for vehicle ID %d ...\n", vehicleID)
			PeriodicPriceUpdateControlProcessVehicle_Tibber(vehicle)
		}
	}
}

func PeriodicPriceUpdateControlProcessVehicle_Tibber(vehicle *Vehicle) {
	priceInfo, err := TibberAPIGetPrices(vehicle.TibberToken)
	if err != nil {
		log.Println(err)
		return
	}
	for _, price := range priceInfo.Today {
		PeriodicPriceUpdateControlProcessPriceInfo_Tibber(vehicle, &price)
	}
	for _, price := range priceInfo.Tomorrow {
		PeriodicPriceUpdateControlProcessPriceInfo_Tibber(vehicle, &price)
	}
}

func PeriodicPriceUpdateControlProcessPriceInfo_Tibber(vehicle *Vehicle, price *GridPrice) {
	ts := price.StartsAt.UTC()
	GetDB().SetTibberPrice(vehicle.ID, ts.Year(), int(ts.Month()), ts.Day(), ts.Hour(), price.Total)
}
