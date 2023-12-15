package main

import (
	"fmt"
	"log"
	"time"
)

func IsCurrentHourUTC(ts *time.Time) bool {
	now := time.Now().UTC()
	if ts.Year() == now.Year() &&
		ts.Month() == now.Month() &&
		ts.Day() == now.Day() &&
		ts.Hour() == now.Hour() {
		return true
	}
	return false
}

func UpdateVehicleDataSaveSoC(authToken string, vehicle *Vehicle) int {
	data, err := TeslaAPIGetVehicleData(authToken, vehicle)
	if err != nil {
		log.Println(err)
		LogChargingEvent(vehicle.ID, LogEventVehicleUpdateData, err.Error())
		return 0
	} else {
		SetVehicleStateSoC(vehicle.ID, data.ChargeState.BatteryLevel)
		LogChargingEvent(vehicle.ID, LogEventVehicleUpdateData, fmt.Sprintf("vehicle SoC updated: %d", data.ChargeState.BatteryLevel))
		return data.ChargeState.BatteryLevel
	}
}
