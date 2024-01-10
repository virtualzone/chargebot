package main

import (
	"fmt"
	"math"
	"time"
)

const MaxVehicleDataUpdateIntervalMinutes int = 15

type ChargeController struct {
	Ticker *time.Ticker
	Time   Time
}

func NewChargeController() *ChargeController {
	return &ChargeController{
		Time: new(RealTime),
	}
}

func (c *ChargeController) Init() {
	c.Ticker = time.NewTicker(time.Minute * 1)
	go func() {
		for {
			<-c.Ticker.C
			c.OnTick()
		}
	}()
}

func (c *ChargeController) OnTick() {
	vehicles := GetDB().GetAllVehicles()
	for _, vehicle := range vehicles {
		c.processVehicle(vehicle)
	}
}

func (c *ChargeController) processVehicle(vehicle *Vehicle) {
	state := GetDB().GetVehicleState(vehicle.ID)
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

	accessToken := GetTeslaAPI().GetOrRefreshAccessToken(vehicle.UserID)

	if accessToken == "" {
		return
	}

	if !vehicle.Enabled && state.Charging != ChargeStateNotCharging {
		// Stop charging if vehicle is still charging but not enabled anymore
		c.stopCharging(accessToken, vehicle)
	} else if vehicle.Enabled && state.Charging == ChargeStateNotCharging {
		// Check if we need to start charging
		c.checkStartCharging(accessToken, vehicle, state)
	} else if vehicle.Enabled && state.Charging != ChargeStateNotCharging {
		// This car is currently charging - check the process
		// only check every 5 minutes to avoid over-adjusting
		if c.Time.UTCNow().Minute()%5 == 0 {
			c.checkChargeProcess(accessToken, vehicle, state)
		}
	}
}

func (c *ChargeController) stopCharging(accessToken string, vehicle *Vehicle) {
	GetTeslaAPI().ChargeStop(accessToken, vehicle)
	GetDB().SetVehicleStateCharging(vehicle.ID, ChargeStateNotCharging)
	GetDB().SetVehicleStateAmps(vehicle.ID, 0)
	GetDB().LogChargingEvent(vehicle.ID, LogEventChargeStop, "smart charging is disabled")
}

func (c *ChargeController) checkTargetState(vehicle *Vehicle, state *VehicleState) (ChargeState, int) {
	targetState := ChargeStateNotCharging
	startCharging, amps := c.checkStartOnSolar(vehicle)
	if startCharging {
		targetState = ChargeStateChargingOnSolar
	} else {
		startCharging, amps = c.checkStartOnTibber(vehicle, state)
		if startCharging {
			targetState = ChargeStateChargingOnGrid
		}
	}
	return targetState, amps
}

func (c *ChargeController) checkStartCharging(accessToken string, vehicle *Vehicle, state *VehicleState) {
	if state.SoC >= vehicle.TargetSoC {
		// nothing to do if target SoC is already reached
		return
	}

	// check if there is a solar surplus
	targetState, amps := c.checkTargetState(vehicle, state)
	if targetState != ChargeStateNotCharging {
		c.activateCharging(accessToken, vehicle, state, amps, targetState)
	}
}

func (c *ChargeController) activateCharging(accessToken string, vehicle *Vehicle, state *VehicleState, amps int, source ChargeState) bool {
	// minimum surplus available, start charging by first waking up the car
	if err := GetTeslaAPI().WakeUpVehicle(accessToken, vehicle); err != nil {
		GetDB().LogChargingEvent(vehicle.ID, LogEventWakeVehicle, "could not wake vehicle: "+err.Error())
		return false
	}
	GetDB().LogChargingEvent(vehicle.ID, LogEventWakeVehicle, "")

	// set the charge limit
	if _, err := GetTeslaAPI().SetChargeLimit(accessToken, vehicle, vehicle.TargetSoC); err != nil {
		GetDB().LogChargingEvent(vehicle.ID, LogEventSetTargetSoC, "could not set target SoC: "+err.Error())
		return false
	}
	GetDB().LogChargingEvent(vehicle.ID, LogEventSetTargetSoC, fmt.Sprintf("target SoC set to %d", vehicle.TargetSoC))

	// set amps to charge
	if _, err := GetTeslaAPI().SetChargeAmps(accessToken, vehicle, amps); err != nil {
		GetDB().LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, "could not set charge amps: "+err.Error())
		return false
	}
	GetDB().LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, fmt.Sprintf("charge amps set to %d", amps))

	// disable scheduled charging
	if _, err := GetTeslaAPI().SetScheduledCharging(accessToken, vehicle, false, 0); err != nil {
		GetDB().LogChargingEvent(vehicle.ID, LogEventSetScheduledCharging, "could not disable scheduled charging: "+err.Error())
		return false
	}
	GetDB().LogChargingEvent(vehicle.ID, LogEventSetScheduledCharging, fmt.Sprintf("disabled scheduled charging"))

	GetTeslaAPI().ChargeStart(accessToken, vehicle)
	GetDB().SetVehicleStateCharging(vehicle.ID, source)
	GetDB().LogChargingEvent(vehicle.ID, LogEventChargeStart, "")

	// charging should start now
	return true
}

func (c *ChargeController) checkStartOnSolar(vehicle *Vehicle) (bool, int) {
	if !vehicle.SurplusCharging {
		return false, 0
	}

	surpluses := GetDB().GetLatestSurplusRecords(vehicle.ID, 1)
	if len(surpluses) == 0 {
		return false, 0
	}

	surplus := surpluses[0]
	// check if this is a recent recording
	now := c.Time.UTCNow()
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

func (c *ChargeController) getEstimatedChargeDurationMinutes(vehicle *Vehicle, state *VehicleState) int {
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

func (c *ChargeController) checkStartOnTibber(vehicle *Vehicle, state *VehicleState) (bool, int) {
	if !vehicle.LowcostCharging {
		return false, 0
	}

	// get upcoming tibber prices sorted by ascending price
	prices := GetDB().GetUpcomingTibberPrices(vehicle.ID, true)
	if len(prices) == 0 {
		return false, 0
	}

	// check if lowest price is above user-defined maximum
	if prices[0].Total*100 > float32(vehicle.MaxPrice) {
		return false, 0
	}

	// check if "now" is below the user-defined maximum
	currentPrice := c.getCurrentTibberPrice(prices)
	if currentPrice == nil {
		return false, 0
	}
	if currentPrice.Total*100 > float32(vehicle.MaxPrice) {
		return false, 0
	}

	// check if current price is lowest of all known prices
	if currentPrice.Total == prices[0].Total {
		return true, vehicle.MaxAmps
	}

	// if not, the current hour may nevertheless be necessary for reach the required charging time
	estimatedChargingTime := c.getEstimatedChargeDurationMinutes(vehicle, state)
	requiredHourBlocks := int(math.Ceil(float64(estimatedChargingTime) / 60))
	for i, price := range prices {
		if i+1 <= requiredHourBlocks {
			now := c.Time.UTCNow()
			if IsCurrentHourUTC(&now, &price.StartsAt) {
				return true, vehicle.MaxAmps
			}
		}
	}

	return false, 0
}

func (c *ChargeController) getCurrentTibberPrice(prices []*TibberPrice) *TibberPrice {
	now := c.Time.UTCNow()
	for _, price := range prices {
		if IsCurrentHourUTC(&now, &price.StartsAt) {
			return price
		}
	}
	return nil
}

func (c *ChargeController) canUpdateVehicleData(vehicleID int) bool {
	event := GetDB().GetLatestChargingEvent(vehicleID, LogEventVehicleUpdateData)
	if event == nil {
		return true
	}
	limit := c.Time.UTCNow().Add(time.Minute * time.Duration(MaxVehicleDataUpdateIntervalMinutes) * -1)
	return event.Timestamp.Before(limit)
}

func (c *ChargeController) minimumChargeTimeReached(vehicle *Vehicle) bool {
	event := GetDB().GetLatestChargingEvent(vehicle.ID, LogEventChargeStart)
	if event == nil {
		return true
	}
	limit := c.Time.UTCNow().Add(time.Minute * time.Duration(vehicle.MinChargeTime) * -1)
	return event.Timestamp.Before(limit)
}

func (c *ChargeController) checkChargeProcess(accessToken string, vehicle *Vehicle, state *VehicleState) {
	// check when vehicle data was last updated
	if c.canUpdateVehicleData(vehicle.ID) {
		state.SoC = UpdateVehicleDataSaveSoC(accessToken, vehicle)
	}

	// if target SoC is reached: stop charging
	if state.SoC >= vehicle.TargetSoC {
		c.stopCharging(accessToken, vehicle)
		return
	}

	// check how the new charging state should be
	targetState, targetAmps := c.checkTargetState(vehicle, state)

	// if minimum charge time is not reached, do nothing
	if !c.minimumChargeTimeReached(vehicle) {
		// ...except when vehicle is charging on solar and amps need to be adjusted
		if state.Charging == ChargeStateChargingOnSolar && targetAmps > 0 && targetAmps != state.Amps {
			if _, err := GetTeslaAPI().SetChargeAmps(accessToken, vehicle, targetAmps); err != nil {
				GetDB().LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, "could not set charge amps: "+err.Error())
			}
			GetDB().LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, fmt.Sprintf("charge amps set to %d", targetAmps))
		}
		return
	}

	// else, check if charging needs to be stopped
	if targetState == ChargeStateNotCharging {
		c.stopCharging(accessToken, vehicle)
	}
}
