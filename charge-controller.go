package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"
)

const MaxVehicleDataUpdateIntervalMinutes int = 15

type ChargeController struct {
	Ticker *time.Ticker
	Time   Time
	Async  bool
}

func NewChargeController() *ChargeController {
	return &ChargeController{
		Time:  new(RealTime),
		Async: true,
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
		if c.Async {
			go c.processVehicle(vehicle)
		} else {
			c.processVehicle(vehicle)
		}
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
	car, err := GetTeslaAPI().InitSession(accessToken, vehicle, false)
	if err != nil {
		GetDB().LogChargingEvent(vehicle.ID, LogEventChargeStop, fmt.Sprintf("could not init session with car: %s", err.Error()))
		return
	}
	if err := GetTeslaAPI().ChargeStop(car); err != nil {
		GetDB().LogChargingEvent(vehicle.ID, LogEventChargeStop, fmt.Sprintf("could not stop charging: %s", err.Error()))
		return
	}
	GetDB().SetVehicleStateCharging(vehicle.ID, ChargeStateNotCharging)
	GetDB().SetVehicleStateAmps(vehicle.ID, 0)
	GetDB().LogChargingEvent(vehicle.ID, LogEventChargeStop, "charging stopped")
}

func (c *ChargeController) checkTargetState(vehicle *Vehicle, state *VehicleState) (ChargeState, int) {
	targetState := ChargeStateNotCharging
	startCharging, amps := c.checkStartOnSolar(vehicle)
	if startCharging {
		targetState = ChargeStateChargingOnSolar
	} else {
		startCharging, amps = c.checkStartOnGrid(vehicle, state)
		if startCharging {
			targetState = ChargeStateChargingOnGrid
		}
	}
	return targetState, amps
}

func (c *ChargeController) isChargingRequired(currentSoC int, targetSoC int) bool {
	return currentSoC < (targetSoC - 1)
}

func (c *ChargeController) checkStartCharging(accessToken string, vehicle *Vehicle, state *VehicleState) {
	if !c.isChargingRequired(state.SoC, vehicle.TargetSoC) {
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
	car, err := GetTeslaAPI().InitSession(accessToken, vehicle, true)
	if err != nil {
		GetDB().LogChargingEvent(vehicle.ID, LogEventWakeVehicle, "could not wake vehicle: "+err.Error())
		return false
	}
	GetDB().LogChargingEvent(vehicle.ID, LogEventWakeVehicle, "")

	// ensure current SoC has not changed in the meantime
	state.SoC = UpdateVehicleDataSaveSoC(accessToken, vehicle)
	if !c.isChargingRequired(state.SoC, vehicle.TargetSoC) {
		GetDB().LogChargingEvent(vehicle.ID, LogEventChargeStart, "charging skipped, target SoC is already reached")
		return false
	}

	// set the charge limit
	if err := GetTeslaAPI().SetChargeLimit(car, vehicle.TargetSoC); err != nil {
		GetDB().LogChargingEvent(vehicle.ID, LogEventSetTargetSoC, "could not set target SoC: "+err.Error())
		return false
	}
	GetDB().LogChargingEvent(vehicle.ID, LogEventSetTargetSoC, fmt.Sprintf("target SoC set to %d", vehicle.TargetSoC))

	// set amps to charge
	if err := GetTeslaAPI().SetChargeAmps(car, amps); err != nil {
		GetDB().LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, "could not set charge amps: "+err.Error())
		return false
	}
	GetDB().LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, fmt.Sprintf("charge amps set to %d", amps))

	GetTeslaAPI().ChargeStart(car)
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

func (c *ChargeController) getUpcomingGridPrices(vehicle *Vehicle) []*GridPrice {
	if vehicle.GridProvider == GridProviderTibber {
		prices := GetDB().GetUpcomingTibberPrices(vehicle.ID, true)
		return prices
	}
	return []*GridPrice{}
}

func (c *ChargeController) checkStartOnGrid_NoDeparturePriceLimit(vehicle *Vehicle, state *VehicleState, prices []*GridPrice) bool {
	// check if lowest price is above user-defined maximum
	if prices[0].Total*100 > float32(vehicle.MaxPrice) {
		return false
	}

	// check if "now" is below the user-defined maximum
	currentPrice := c.getCurrentGridPrice(prices)
	if currentPrice == nil {
		return false
	}
	if currentPrice.Total*100 > float32(vehicle.MaxPrice) {
		return false
	}

	// check if current price is lowest of all known prices
	if currentPrice.Total == prices[0].Total {
		return true
	}

	// if not, the current hour may nevertheless be necessary for reach the required charging time
	estimatedChargingTime := c.getEstimatedChargeDurationMinutes(vehicle, state)
	requiredHourBlocks := int(math.Ceil(float64(estimatedChargingTime) / 60))
	for i, price := range prices {
		if i+1 <= requiredHourBlocks {
			now := c.Time.UTCNow()
			if IsCurrentHourUTC(&now, &price.StartsAt) {
				if price.Total*100 <= float32(vehicle.MaxPrice) {
					return true
				}
			}
		}
	}

	return false
}

func (c *ChargeController) getNextDeparture(vehicle *Vehicle) (*time.Time, error) {
	now := c.Time.UTCNow()
	curWeekday := now.Weekday()
	if curWeekday == time.Sunday {
		curWeekday = 7
	}
	timeTokens, err := AtoiArray(strings.Split(vehicle.DepartTime, ":"))
	if err != nil {
		return nil, err
	}
	dayTokens, err := AtoiArray(strings.Split(vehicle.DepartDays, ""))
	if err != nil {
		return nil, err
	}
	sort.Ints(dayTokens)
	// check if current day is next departure
	for _, day := range dayTokens {
		if day == int(curWeekday) {
			if timeTokens[0] > now.Hour() {
				res := time.Date(now.Year(), now.Month(), now.Day(), timeTokens[0], timeTokens[1], 0, 0, now.Location())
				return &res, nil
			}
		}
	}
	// else, use next day
	for _, day := range dayTokens {
		if day > int(curWeekday) {
			res := time.Date(now.Year(), now.Month(), now.Day(), timeTokens[0], timeTokens[1], 0, 0, now.Location())
			res = res.AddDate(0, 0, day-int(curWeekday))
			return &res, nil
		}
	}
	// else, use first day in list
	day := dayTokens[0]
	res := time.Date(now.Year(), now.Month(), now.Day(), timeTokens[0], timeTokens[1], 0, 0, now.Location())
	res = res.AddDate(0, 0, 7-int(curWeekday)+day)
	return &res, nil
}

func (c *ChargeController) checkStartOnGrid_DepartureNoPriceLimit(vehicle *Vehicle, state *VehicleState, prices []*GridPrice) bool {
	departure, err := c.getNextDeparture(vehicle)
	if err != nil {
		log.Printf("could not get next departure date for vehicle %d: %s\n", vehicle.ID, err.Error())
		return false
	}
	pricesFiltered := c.getGridPricesBefore(prices, *departure)

	estimatedChargingTime := c.getEstimatedChargeDurationMinutes(vehicle, state)
	requiredHourBlocks := int(math.Ceil(float64(estimatedChargingTime) / 60))

	timeUntilDeparture := departure.Sub(c.Time.UTCNow())
	// do nothing if we don't know the prices valid until departure
	if !c.containsPricesUntilDeparture(pricesFiltered, *departure) {
		// but only if the time until departure is at least twice the estimated charging time
		if timeUntilDeparture.Minutes() >= float64(estimatedChargingTime)*2 {
			return false
		}
	}

	for i, price := range pricesFiltered {
		if i+1 <= requiredHourBlocks {
			now := c.Time.UTCNow()
			if IsCurrentHourUTC(&now, &price.StartsAt) {
				return true
			}
		}
	}

	return false
}

func (c *ChargeController) checkStartOnGrid_DepartureWithPriceLimit(vehicle *Vehicle, state *VehicleState, prices []*GridPrice) bool {
	departure, err := c.getNextDeparture(vehicle)
	if err != nil {
		log.Printf("could not get next departure date for vehicle %d: %s\n", vehicle.ID, err.Error())
		return false
	}
	pricesFiltered := c.getGridPricesBefore(prices, *departure)

	// check if lowest price is above user-defined maximum
	if pricesFiltered[0].Total*100 > float32(vehicle.MaxPrice) {
		return false
	}

	// check if "now" is below the user-defined maximum
	currentPrice := c.getCurrentGridPrice(pricesFiltered)
	if currentPrice == nil {
		return false
	}
	if currentPrice.Total*100 > float32(vehicle.MaxPrice) {
		return false
	}

	// check if current price is lowest of all known prices
	if currentPrice.Total == pricesFiltered[0].Total {
		return true
	}

	// if not, the current hour may nevertheless be necessary for reach the required charging time
	//timeUntilDeparture := departure.Sub(c.Time.UTCNow())
	estimatedChargingTime := c.getEstimatedChargeDurationMinutes(vehicle, state)
	requiredHourBlocks := int(math.Ceil(float64(estimatedChargingTime) / 60))
	for i, price := range pricesFiltered {
		if i+1 <= requiredHourBlocks {
			now := c.Time.UTCNow()
			if IsCurrentHourUTC(&now, &price.StartsAt) {
				if price.Total*100 <= float32(vehicle.MaxPrice) {
					return true
				}
			}
		}
	}

	for i, price := range pricesFiltered {
		if i+1 <= requiredHourBlocks {
			now := c.Time.UTCNow()
			if IsCurrentHourUTC(&now, &price.StartsAt) {
				return true
			}
		}
	}

	return false
}

func (c *ChargeController) checkStartOnGrid(vehicle *Vehicle, state *VehicleState) (bool, int) {
	if !vehicle.LowcostCharging {
		return false, 0
	}

	// get upcoming grid prices sorted by ascending price
	prices := c.getUpcomingGridPrices(vehicle)
	if len(prices) == 0 {
		return false, 0
	}

	// check whether to start charging depending on grid strategy
	res := false
	switch vehicle.GridStrategy {
	case GridStrategyNoDeparturePriceLimit:
		res = c.checkStartOnGrid_NoDeparturePriceLimit(vehicle, state, prices)
	case GridStrategyDepartureNoPriceLimit:
		res = c.checkStartOnGrid_DepartureNoPriceLimit(vehicle, state, prices)
	case GridStrategyDepartureWithPriceLimit:
		res = c.checkStartOnGrid_DepartureWithPriceLimit(vehicle, state, prices)
	}

	if res {
		return true, vehicle.MaxAmps
	}

	return false, 0
}

func (c *ChargeController) getGridPricesBefore(prices []*GridPrice, limit time.Time) []*GridPrice {
	res := []*GridPrice{}
	for _, price := range prices {
		if price.StartsAt.Before(limit) {
			res = append(res, price)
		}
	}
	return res
}

func (c *ChargeController) containsPricesUntilDeparture(prices []*GridPrice, departure time.Time) bool {
	for _, price := range prices {
		if price.StartsAt.Equal(departure) || price.StartsAt.After(departure) || departure.Sub(price.StartsAt).Minutes() <= 60 {
			return true
		}
	}
	return false
}

func (c *ChargeController) getCurrentGridPrice(prices []*GridPrice) *GridPrice {
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
			car, err := GetTeslaAPI().InitSession(accessToken, vehicle, false)
			if err != nil {
				GetDB().LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, fmt.Sprintf("could not init session with car: %s", err.Error()))
			} else {
				if err := GetTeslaAPI().SetChargeAmps(car, targetAmps); err != nil {
					GetDB().LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, "could not set charge amps: "+err.Error())
				}
				GetDB().LogChargingEvent(vehicle.ID, LogEventSetChargingAmps, fmt.Sprintf("charge amps set to %d", targetAmps))
			}
		}
		return
	}

	// else, check if charging needs to be stopped
	if targetState == ChargeStateNotCharging {
		c.stopCharging(accessToken, vehicle)
	}
}
