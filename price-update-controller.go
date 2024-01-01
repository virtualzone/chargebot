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
	// First, care about the vehicles that don't even have prices for today
	// Limit to 45 vehicles so we don't exceed the API limits
	l := GetDB().GetVehicleIDsWithTibberTokenWithoutPricesForToday(45)
	for _, vehicleID := range l {
		vehicle := GetDB().GetVehicleByID(vehicleID)
		log.Printf("Updating today's Tibber prices for vehicle ID %d ...\n", vehicleID)
		PeriodicPriceUpdateControlProcessVehicle(vehicle)
	}

	now := time.Now().UTC()
	if now.Hour() > 12 {
		// Next, if it's past 13:00 GMT, handle the vehicle's without prices for tomorrow
		l := GetDB().GetVehicleIDsWithTibberTokenWithoutPricesForTomorrow(45)
		for _, vehicleID := range l {
			vehicle := GetDB().GetVehicleByID(vehicleID)
			log.Printf("Updating tomorrow's Tibber prices for vehicle ID %d ...\n", vehicleID)
			PeriodicPriceUpdateControlProcessVehicle(vehicle)
		}
	}
}

func PeriodicPriceUpdateControlProcessVehicle(vehicle *Vehicle) {
	priceInfo, err := TibberAPIGetPrices(vehicle.TibberToken)
	if err != nil {
		log.Println(err)
		return
	}
	for _, price := range priceInfo.Today {
		PeriodicPriceUpdateControlProcessPriceInfo(vehicle, &price)
	}
	for _, price := range priceInfo.Tomorrow {
		PeriodicPriceUpdateControlProcessPriceInfo(vehicle, &price)
	}
}

func PeriodicPriceUpdateControlProcessPriceInfo(vehicle *Vehicle, price *TibberPrice) {
	ts := price.StartsAt.UTC()
	GetDB().SetTibberPrice(vehicle.ID, ts.Year(), int(ts.Month()), ts.Day(), ts.Hour(), price.Total)
}
