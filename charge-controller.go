package main

import (
	"log"
	"time"
)

var Ticker *time.Ticker = nil

func InitPeriodicChargeControl() {
	Ticker = time.NewTicker(time.Minute * 1)
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
		PeriodicChargeControlProcessVehicle(vehicle)
	}
}

func PeriodicChargeControlProcessVehicle(vehicle *Vehicle) {
	state := GetVehicleState(vehicle.ID)
	if state == nil {
		// no state yet, so nothing to do
		return
	}

	if !state.PluggedIn {
		// nothing to do for an unplugged vehicle
		return
	}

	if !vehicle.Enabled && !state.Charging {
		// nothing to do for a disabled vehicle which is not charging
		return
	}

	accessToken := TeslaAPIGetOrRefreshAccessToken(vehicle.UserID)
	if accessToken == "" {
		return
	}

	if !vehicle.Enabled && state.Charging {
		// Stop charging if vehicle is still charging but not enabled anymore
		ChargeControlStopCharging(accessToken, vehicle)
	} else if vehicle.Enabled && !state.Charging {
		// Check if we need to start charging
		ChargeControlCheckStartCharging(accessToken, vehicle, state)
	} else if vehicle.Enabled && state.Charging {
		// This car is currently charging - check the process
		ChargeControlCheckChargeProcess(accessToken, vehicle, state)
	}
}

func ChargeControlStopCharging(accessToken string, vehicle *Vehicle) {
	TeslaAPIChargeStop(accessToken, vehicle)
	SetVehicleStateCharging(vehicle.ID, false)
	LogChargingEvent(vehicle.ID, LogEventChargeStop, "smart charging is disabled")
}

func ChargeControlCheckStartCharging(accessToken string, vehicle *Vehicle, state *VehicleState) {
	if state.SoC >= vehicle.TargetSoC {
		// nothing to do if target SoC is already reached
		return
	}

	// todo
}

func ChargeControlCheckChargeProcess(accessToken string, vehicle *Vehicle, state *VehicleState) {
	// get current SoC
	data, err := TeslaAPIGetVehicleData(accessToken, vehicle)
	if err != nil {
		log.Println(err)
	} else {
		SetVehicleStateSoC(vehicle.ID, data.ChargeState.BatteryLevel)
	}
}
