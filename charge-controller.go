package main

import (
	"fmt"
	"log"
	"math"
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

	// check if there is a solar surplus
	startedOnSolar := ChargeControlCheckStartOnSolar(accessToken, vehicle, state)
	if startedOnSolar {
		return
	}

	startedOnTibber := ChargeControlCheckStartOnTibber(accessToken, vehicle, state)
	if startedOnTibber {
		return
	}
}

func ChargeControlActivateCharging(accessToken string, vehicle *Vehicle, state *VehicleState, amps int) bool {
	// minimum surplus available, start charging by first waking up the car
	if err := TeslaAPIWakeUpVehicle(accessToken, vehicle); err != nil {
		LogChargingEvent(vehicle.ID, LogEventWakeVehicle, "could not wake vehicle: "+err.Error())
		return false
	}
	LogChargingEvent(vehicle.ID, LogEventWakeVehicle, "")

	// set the charge limit
	if _, err := TeslaAPISetChargeLimit(accessToken, vehicle, vehicle.TargetSoC); err != nil {
		LogChargingEvent(vehicle.ID, LogEventSetTargetSoC, "could not set target SoC: "+err.Error())
		return false
	}
	LogChargingEvent(vehicle.ID, LogEventSetTargetSoC, fmt.Sprintf("target SoC set to %d", vehicle.TargetSoC))

	// set amps to charge
	if _, err := TeslaAPISetChargeAmps(accessToken, vehicle, amps); err != nil {
		LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, "could not set charge amps: "+err.Error())
		return false
	}
	LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, fmt.Sprintf("charge amps set to %d", amps))

	// disable scheduled charging
	if _, err := TeslaAPISetScheduledCharging(accessToken, vehicle, false, 0); err != nil {
		LogChargingEvent(vehicle.ID, LogEventSetScheduledCharging, "could not disable scheduled charging: "+err.Error())
		return false
	}
	LogChargingEvent(vehicle.ID, LogEventSetScheduledCharging, fmt.Sprintf("disabled scheduled charging"))

	// charging should start now
	return true
}

func ChargeControlCheckStartOnSolar(accessToken string, vehicle *Vehicle, state *VehicleState) bool {
	surpluses := GetLatestSurplusRecords(vehicle.ID, 1)
	if len(surpluses) == 0 {
		return false
	}

	surplus := surpluses[0]
	// check if this is a recent recording
	now := time.Now().UTC()
	if surplus.Timestamp.Before(now.Add(-10 * time.Minute)) {
		return false
	}

	// check if surplus minimum is reached
	if surplus.SurplusWatts < vehicle.MinSurplus {
		return false
	}

	// determine amps to charge
	amps := int(math.Round(float64(surplus.SurplusWatts) / 230.0 / float64(vehicle.NumPhases)))
	if amps == 0 {
		return false
	}
	if amps > vehicle.MaxAmps {
		amps = vehicle.MaxAmps
	}

	return ChargeControlActivateCharging(accessToken, vehicle, state, amps)
}

func ChargeControlGetEstimatedChargeDurationMinutes(vehicle *Vehicle, state *VehicleState) int {
	percentToCharge := vehicle.TargetSoC - state.SoC
	wattsPerHour := vehicle.MaxAmps * vehicle.NumPhases * 230
	batteryCapacitykWh := 100 // assume large battery for now
	estimatedHoursToCharge := float64(percentToCharge) / 100 * float64(batteryCapacitykWh) / (float64(wattsPerHour) / 1000)
	return int(math.Round(estimatedHoursToCharge * 60))
}

func ChargeControlCheckStartOnTibber(accessToken string, vehicle *Vehicle, state *VehicleState) bool {
	prices := GetUpcomingTibberPrices(vehicle.ID)
	if len(prices) == 0 {
		return false
	}

	//estimatedMin

	return ChargeControlActivateCharging(accessToken, vehicle, state, vehicle.MaxAmps)
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
