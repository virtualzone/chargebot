package main

import (
	"fmt"
	"log"
	"math"
	"time"
)

var Ticker *time.Ticker = nil

const MaxVehicleDataUpdateIntervalMinutes int = 15

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

	if !vehicle.Enabled && state.Charging == ChargeStateNotCharging {
		// nothing to do for a disabled vehicle which is not charging
		return
	}

	accessToken := TeslaAPIGetOrRefreshAccessToken(vehicle.UserID)
	if accessToken == "" {
		return
	}

	if !vehicle.Enabled && state.Charging != ChargeStateNotCharging {
		// Stop charging if vehicle is still charging but not enabled anymore
		ChargeControlStopCharging(accessToken, vehicle)
	} else if vehicle.Enabled && state.Charging == ChargeStateNotCharging {
		// Check if we need to start charging
		ChargeControlCheckStartCharging(accessToken, vehicle, state)
	} else if vehicle.Enabled && state.Charging != ChargeStateNotCharging {
		// This car is currently charging - check the process
		// only check every 5 minutes to avoid over-adjusting
		if time.Now().Minute()%5 == 0 {
			ChargeControlCheckChargeProcess(accessToken, vehicle, state)
		}
	}
}

func ChargeControlStopCharging(accessToken string, vehicle *Vehicle) {
	TeslaAPIChargeStop(accessToken, vehicle)
	SetVehicleStateCharging(vehicle.ID, ChargeStateNotCharging)
	SetVehicleStateAmps(vehicle.ID, 0)
	LogChargingEvent(vehicle.ID, LogEventChargeStop, "smart charging is disabled")
}

func ChargeControlCheckTargetState(vehicle *Vehicle, state *VehicleState) (ChargeState, int) {
	targetState := ChargeStateNotCharging
	startCharging, amps := ChargeControlCheckStartOnSolar(vehicle)
	if startCharging {
		targetState = ChargeStateChargingOnSolar
	} else {
		startCharging, amps = ChargeControlCheckStartOnTibber(vehicle, state)
		if startCharging {
			targetState = ChargeStateChargingOnGrid
		}
	}
	return targetState, amps
}

func ChargeControlCheckStartCharging(accessToken string, vehicle *Vehicle, state *VehicleState) {
	if state.SoC >= vehicle.TargetSoC {
		// nothing to do if target SoC is already reached
		return
	}

	// check if there is a solar surplus
	targetState, amps := ChargeControlCheckTargetState(vehicle, state)
	if targetState != ChargeStateNotCharging {
		ChargeControlActivateCharging(accessToken, vehicle, state, amps, targetState)
	}
}

func ChargeControlActivateCharging(accessToken string, vehicle *Vehicle, state *VehicleState, amps int, source ChargeState) bool {
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

	SetVehicleStateCharging(vehicle.ID, source)

	// charging should start now
	return true
}

func ChargeControlCheckStartOnSolar(vehicle *Vehicle) (bool, int) {
	if !vehicle.SurplusCharging {
		return false, 0
	}

	surpluses := GetLatestSurplusRecords(vehicle.ID, 1)
	if len(surpluses) == 0 {
		return false, 0
	}

	surplus := surpluses[0]
	// check if this is a recent recording
	now := time.Now().UTC()
	if surplus.Timestamp.Before(now.Add(-10 * time.Minute)) {
		return false, 0
	}

	// check if there is any surplus
	if surplus.SurplusWatts <= 0 {
		return false, 0
	}

	// check if surplus minimum is reached
	if surplus.SurplusWatts < vehicle.MinSurplus {
		return false, 0
	}

	// determine amps to charge
	amps := int(math.Round(float64(surplus.SurplusWatts) / 230.0 / float64(vehicle.NumPhases)))
	if amps == 0 {
		return false, 0
	}
	if amps > vehicle.MaxAmps {
		amps = vehicle.MaxAmps
	}

	return true, amps
}

func ChargeControlGetEstimatedChargeDurationMinutes(vehicle *Vehicle, state *VehicleState) int {
	percentToCharge := vehicle.TargetSoC - state.SoC
	wattsPerHour := vehicle.MaxAmps * vehicle.NumPhases * 230
	batteryCapacitykWh := 100 // assume large battery for now
	estimatedHoursToCharge := float64(percentToCharge) / 100 * float64(batteryCapacitykWh) / (float64(wattsPerHour) / 1000)
	res := int(math.Round(estimatedHoursToCharge * 60))
	if res < 0 {
		return 0
	}
	return res
}

func ChargeControlCheckStartOnTibber(vehicle *Vehicle, state *VehicleState) (bool, int) {
	if !vehicle.LowcostCharging {
		return false, 0
	}

	// get upcoming tibber prices sorted by ascending price
	prices := GetUpcomingTibberPrices(vehicle.ID, true)
	if len(prices) == 0 {
		return false, 0
	}

	// check if lowest price is above user-defined maximum
	if prices[0].Total*100 > float32(vehicle.MaxPrice) {
		return false, 0
	}

	// check if "now" is below the user-defined maximum
	currentPrice := ChargeControlGetCurrentTibberPrice(prices)
	if currentPrice.Total*100 > float32(vehicle.MaxPrice) {
		return false, 0
	}

	// check if current price is lowest of all known prices
	if currentPrice.Total == prices[0].Total {
		return true, vehicle.MaxAmps
	}

	// if not, the current hour may nevertheless be necessary for reach the required charging time
	estimatedChargingTime := ChargeControlGetEstimatedChargeDurationMinutes(vehicle, state)
	requiredHourBlocks := int(math.Ceil(float64(estimatedChargingTime) / 60))
	for i, price := range prices {
		if i+1 <= requiredHourBlocks {
			if IsCurrentHourUTC(&price.StartsAt) {
				return true, vehicle.MaxAmps
			}
		}
	}

	return false, 0
}

func ChargeControlGetCurrentTibberPrice(prices []*TibberPrice) *TibberPrice {
	for _, price := range prices {
		if IsCurrentHourUTC(&price.StartsAt) {
			return price
		}
	}
	return nil
}

func ChargeControlCanUpdateVehicleData(vehicleID int) bool {
	event := GetLatestChargingEvent(vehicleID, LogEventVehicleUpdateData)
	if event == nil {
		return true
	}
	limit := time.Now().UTC().Add(time.Minute * time.Duration(MaxVehicleDataUpdateIntervalMinutes) * -1)
	return event.Timestamp.Before(limit)
}

func ChargeControlMinimumChargeTimeReached(vehicle *Vehicle) bool {
	event := GetLatestChargingEvent(vehicle.ID, LogEventChargeStart)
	if event == nil {
		return true
	}
	limit := time.Now().UTC().Add(time.Minute * time.Duration(vehicle.MinChargeTime) * -1)
	return event.Timestamp.Before(limit)
}

func ChargeControlCheckChargeProcess(accessToken string, vehicle *Vehicle, state *VehicleState) {
	// check when vehicle data was last updated
	if ChargeControlCanUpdateVehicleData(vehicle.ID) {
		state.SoC = UpdateVehicleDataSaveSoC(accessToken, vehicle)
	}

	// if target SoC is reached: stop charging
	if state.SoC >= vehicle.TargetSoC {
		ChargeControlStopCharging(accessToken, vehicle)
		return
	}

	// check how the new charging state should be
	targetState, targetAmps := ChargeControlCheckTargetState(vehicle, state)

	// if minimum charge time is not reached, do nothing
	if !ChargeControlMinimumChargeTimeReached(vehicle) {
		// ...except when vehicle is charging on solar and amps need to be adjusted
		if state.Charging == ChargeStateChargingOnSolar && targetAmps > 0 && targetAmps != state.Amps {
			if _, err := TeslaAPISetChargeAmps(accessToken, vehicle, targetAmps); err != nil {
				LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, "could not set charge amps: "+err.Error())
			}
			LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, fmt.Sprintf("charge amps set to %d", targetAmps))
		}
		return
	}

	// else, check if charging needs to be stopped
	if targetState == ChargeStateNotCharging {
		ChargeControlStopCharging(accessToken, vehicle)
	}
}
