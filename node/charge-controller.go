package main

import (
	"fmt"
	"log"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

const MaxVehicleDataUpdateIntervalMinutes int = 5
const MaxChargeStartFailCounts int = 10

var DelayBetweenAPICommands time.Duration = time.Second * 2

type ChargeController struct {
	Ticker               *time.Ticker
	Time                 Time
	Async                bool
	ChargeStartFailCount int
	inTick               []string
	inTickMutex          sync.Mutex
}

func NewChargeController() *ChargeController {
	return &ChargeController{
		Time:                 new(RealTime),
		Async:                true,
		ChargeStartFailCount: 0,
		inTick:               []string{},
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
	permanentError := (GetDB().GetSetting(SettingsPermanentError) == "1")
	if permanentError {
		log.Println("ACTION REQUIRED: Permanent error after recurring charge failures. Check Web UI to resolve.")
		return
	}
	vehicles := GetDB().GetVehicles()
	for _, vehicle := range vehicles {
		if c.Async {
			go c.processVehicle(vehicle)
		} else {
			c.processVehicle(vehicle)
		}
	}
}

func (c *ChargeController) setInTick(vin string) {
	c.inTickMutex.Lock()
	defer c.inTickMutex.Unlock()
	if !slices.Contains(c.inTick, vin) {
		c.inTick = append(c.inTick, vin)
	}
}

func (c *ChargeController) unsetInTick(vin string) {
	c.inTickMutex.Lock()
	defer c.inTickMutex.Unlock()
	idx := slices.Index(c.inTick, vin)
	if idx > -1 {
		c.inTick = slices.Delete(c.inTick, idx, 1)
	}
}

func (c *ChargeController) isInTick(vin string) bool {
	return slices.Contains(c.inTick, vin)
}

func (c *ChargeController) processVehicle(vehicle *Vehicle) {
	state := GetDB().GetVehicleState(vehicle.VIN)
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

	if c.isInTick(vehicle.VIN) {
		// skip this round if vehicle is still processing
		return
	}

	c.setInTick(vehicle.VIN)
	defer c.unsetInTick(vehicle.VIN)

	if !vehicle.Enabled && state.Charging != ChargeStateNotCharging {
		// Stop charging if vehicle is still charging but not enabled anymore
		c.stopCharging(vehicle, state)
	} else if !vehicle.SurplusCharging && state.Charging == ChargeStateChargingOnSolar {
		// Stop charging if vehicle is still charging on solar but surplus charging is not enabled anymore
		c.stopCharging(vehicle, state)
	} else if !vehicle.LowcostCharging && state.Charging == ChargeStateChargingOnGrid {
		// Stop charging if vehicle is still charging on grid but grid charging is not enabled anymore
		c.stopCharging(vehicle, state)
	} else if vehicle.Enabled && state.Charging == ChargeStateNotCharging {
		// Check if we need to start charging
		c.checkStartCharging(vehicle, state)
	} else if vehicle.Enabled && state.Charging != ChargeStateNotCharging {
		// This car is currently charging - check the process
		c.checkChargeProcess(vehicle, state)
	}
}

func (c *ChargeController) stopCharging(vehicle *Vehicle, state *VehicleState) {
	err := GetTeslaAPI().Wakeup(vehicle.VIN)
	if err != nil {
		GetDB().LogChargingEvent(vehicle.VIN, LogEventChargeStop, fmt.Sprintf("could not init session with car: %s", err.Error()))
		return
	}

	time.Sleep(DelayBetweenAPICommands) // delay

	if err := GetTeslaAPI().ChargeStop(vehicle.VIN); err != nil {
		GetDB().LogChargingEvent(vehicle.VIN, LogEventChargeStop, fmt.Sprintf("could not stop charging: %s", err.Error()))
		return
	}

	GetDB().SetVehicleStateCharging(vehicle.VIN, ChargeStateNotCharging)
	GetDB().LogChargingEvent(vehicle.VIN, LogEventChargeStop, "charging stopped")

	SendPushNotification(fmt.Sprintf("%s stopped charging at %d %% SoC.", vehicle.DisplayName, state.SoC))
}

func (c *ChargeController) checkTargetState(vehicle *Vehicle, state *VehicleState) (ChargeState, int) {
	targetState := ChargeStateNotCharging
	startCharging, amps := c.checkStartOnSolar(vehicle, state)
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

func (c *ChargeController) checkStartCharging(vehicle *Vehicle, state *VehicleState) {
	if !c.isChargingRequired(state.SoC, vehicle.TargetSoC) {
		// nothing to do if target SoC is already reached
		return
	}

	// check if there is a solar surplus
	targetState, amps := c.checkTargetState(vehicle, state)
	if targetState != ChargeStateNotCharging {
		c.activateCharging(vehicle, state, amps, targetState)
	}
}

func (c *ChargeController) activateCharging(vehicle *Vehicle, state *VehicleState, amps int, source ChargeState) bool {
	if c.ChargeStartFailCount > 0 {
		if c.ChargeStartFailCount >= MaxChargeStartFailCounts {
			log.Printf("Activate charging failed for %d times, giving up and setting permanent error\n", c.ChargeStartFailCount)
			SendPushNotification(fmt.Sprintf("ACTION REQUIRED: Activate charging failed for %d times, giving up and setting permanent error. Resolve issue and release permanent error in Web UI.", c.ChargeStartFailCount))
			GetDB().SetSetting(SettingsPermanentError, "1")
			c.ChargeStartFailCount = 0
			return false
		}
		minWaitTime := time.Minute * time.Duration(c.ChargeStartFailCount*5)
		lastAttempt := GetDB().GetLatestChargingEvent(vehicle.VIN, LogEventChargeStart)
		if lastAttempt != nil {
			if lastAttempt.Timestamp.After(c.Time.UTCNow().Add(minWaitTime * -1)) {
				return false
			}
		}
		log.Printf("This is attempt %d to activate charging after previous errors\n", c.ChargeStartFailCount)
	}

	err := GetTeslaAPI().Wakeup(vehicle.VIN)
	if err != nil {
		GetDB().LogChargingEvent(vehicle.VIN, LogEventWakeVehicle, "could not wake vehicle: "+err.Error())
		return false
	}
	GetDB().LogChargingEvent(vehicle.VIN, LogEventWakeVehicle, "")

	time.Sleep(DelayBetweenAPICommands) // delay

	// set the charge limit
	if state.ChargeLimit != vehicle.TargetSoC {
		if err := GetTeslaAPI().SetChargeLimit(vehicle.VIN, vehicle.TargetSoC); err != nil {
			GetDB().LogChargingEvent(vehicle.VIN, LogEventSetTargetSoC, "could not set target SoC: "+err.Error())
			return false
		}
		GetDB().LogChargingEvent(vehicle.VIN, LogEventSetTargetSoC, fmt.Sprintf("target SoC set to %d", vehicle.TargetSoC))
		time.Sleep(DelayBetweenAPICommands) // delay
	}

	// set amps to charge
	if state.Amps != amps {
		if err := GetTeslaAPI().SetChargeAmps(vehicle.VIN, amps); err != nil {
			GetDB().LogChargingEvent(vehicle.VIN, LogEventSetChargingAmps, "could not set charge amps: "+err.Error())
			return false
		}
		GetDB().LogChargingEvent(vehicle.VIN, LogEventSetChargingAmps, fmt.Sprintf("charge amps set to %d", amps))
		time.Sleep(DelayBetweenAPICommands) // delay
	}

	if err := GetTeslaAPI().ChargeStart(vehicle.VIN); err != nil {
		GetDB().LogChargingEvent(vehicle.VIN, LogEventChargeStart, "could not start charging: "+err.Error())
		c.ChargeStartFailCount++
		return false
	}
	c.ChargeStartFailCount = 0
	GetDB().LogChargingEvent(vehicle.VIN, LogEventChargeStart, "")

	GetDB().SetVehicleStateAmps(vehicle.VIN, amps)
	GetDB().SetVehicleStateCharging(vehicle.VIN, source)

	sourceText := "solar power"
	if source == ChargeStateChargingOnGrid {
		sourceText = "grid"
	}
	SendPushNotification(fmt.Sprintf("%s started charging on %s with %d amps at %d %% SoC.", vehicle.DisplayName, sourceText, amps, state.SoC))

	// charging should start now
	return true
}

func (c *ChargeController) checkStartOnSolar(vehicle *Vehicle, state *VehicleState) (bool, int) {
	if !vehicle.SurplusCharging {
		return false, 0
	}

	surplus := c.getActualSurplus(vehicle, state)
	if surplus <= 0 {
		LogDebug(fmt.Sprintf("checkStartOnSolar() - no current surplus for vehicle %s", vehicle.VIN))
		return false, 0
	}

	// check if surplus minimum is reached
	if surplus < vehicle.MinSurplus {
		LogDebug(fmt.Sprintf("checkStartOnSolar() - too low surplus %d for vehicle %s", surplus, vehicle.VIN))
		return false, 0
	}

	// determine amps to charge
	amps := int(math.Floor(float64(surplus) / 230.0 / float64(vehicle.NumPhases)))
	if amps == 0 {
		return false, 0
	}
	if amps > vehicle.MaxAmps {
		amps = vehicle.MaxAmps
	}
	LogDebug(fmt.Sprintf("checkStartOnSolar() - encourage %d amps for vehicle %s", amps, vehicle.VIN))

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
		prices := GetDB().GetUpcomingTibberPrices(vehicle.VIN, true)
		return prices
	}
	return []*GridPrice{}
}

func (c *ChargeController) checkStartOnGrid_NoDeparturePriceLimit(vehicle *Vehicle, state *VehicleState, prices []*GridPrice) bool {
	now := c.Time.UTCNow()
	if GetDB().IsSelectedGridHourblock(vehicle.VIN, now.Year(), int(now.Month()), now.Day(), now.Hour()) {
		return true
	}

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
					GetDB().RecordSelectedGridHourblock(vehicle.VIN, now.Year(), int(now.Month()), now.Day(), now.Hour())
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
		log.Printf("could not get next departure date for vehicle %s: %s\n", vehicle.VIN, err.Error())
		return false
	}

	now := c.Time.UTCNow()
	if GetDB().IsSelectedGridHourblock(vehicle.VIN, now.Year(), int(now.Month()), now.Day(), now.Hour()) {
		return true
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
			if IsCurrentHourUTC(&now, &price.StartsAt) {
				GetDB().RecordSelectedGridHourblock(vehicle.VIN, now.Year(), int(now.Month()), now.Day(), now.Hour())
				return true
			}
		}
	}

	return false
}

func (c *ChargeController) checkStartOnGrid_DepartureWithPriceLimit(vehicle *Vehicle, state *VehicleState, prices []*GridPrice) bool {
	departure, err := c.getNextDeparture(vehicle)
	if err != nil {
		log.Printf("could not get next departure date for vehicle %s: %s\n", vehicle.VIN, err.Error())
		return false
	}

	now := c.Time.UTCNow()
	if GetDB().IsSelectedGridHourblock(vehicle.VIN, now.Year(), int(now.Month()), now.Day(), now.Hour()) {
		return true
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
			if IsCurrentHourUTC(&now, &price.StartsAt) {
				if price.Total*100 <= float32(vehicle.MaxPrice) {
					GetDB().RecordSelectedGridHourblock(vehicle.VIN, now.Year(), int(now.Month()), now.Day(), now.Hour())
					return true
				}
			}
		}
	}

	for i, price := range pricesFiltered {
		if i+1 <= requiredHourBlocks {
			if IsCurrentHourUTC(&now, &price.StartsAt) {
				GetDB().RecordSelectedGridHourblock(vehicle.VIN, now.Year(), int(now.Month()), now.Day(), now.Hour())
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

func (c *ChargeController) minimumChargeTimeReached(vehicle *Vehicle, state *VehicleState) bool {
	event := GetDB().GetLatestChargingEvent(vehicle.VIN, LogEventChargeStart)
	if event == nil {
		return true
	}
	if state.Charging == ChargeStateChargingOnSolar {
		limit := c.Time.UTCNow().Add(time.Minute * time.Duration(vehicle.MinChargeTime) * -1)
		return event.Timestamp.Before(limit)
	}
	return true
}

func (c *ChargeController) canAdjustSolarAmps(vehicle *Vehicle) bool {
	surpluses := GetDB().GetLatestSurplusRecords(1)
	if len(surpluses) == 0 {
		return false
	}
	surplus := surpluses[0]
	var latest *time.Time = nil
	ampsEvent := GetDB().GetLatestChargingEvent(vehicle.VIN, LogEventSetChargingAmps)
	if ampsEvent != nil {
		latest = &ampsEvent.Timestamp
	}
	startEvent := GetDB().GetLatestChargingEvent(vehicle.VIN, LogEventChargeStart)
	if startEvent != nil {
		if (latest == nil) || (startEvent.Timestamp.After(*latest)) {
			latest = &startEvent.Timestamp
		}
	}
	if latest == nil {
		return false
	}
	diff := surplus.Timestamp.Sub(*latest)
	return diff.Minutes() >= 5 // max. every 5 mins
}

func (c *ChargeController) getActualSurplus(vehicle *Vehicle, state *VehicleState) int {
	numSamples := 2
	now := c.Time.UTCNow()
	surpluses := GetDB().GetLatestSurplusRecords(numSamples)
	if len(surpluses) == 0 {
		return -1
	}
	// if not charging on solar yet, all samples must be above threshold
	if state.Charging != ChargeStateChargingOnSolar {
		res := 0
		allAbove := true
		for _, surplus := range surpluses {
			if surplus.Timestamp.After(now.Add(-5 * time.Minute)) {
				if state.Charging == ChargeStateChargingOnSolar {
					surplus.SurplusWatts += (state.Amps * 230 * vehicle.NumPhases)
				}
				if surplus.SurplusWatts >= vehicle.MinSurplus {
					if surplus.SurplusWatts > res && allAbove {
						res = surplus.SurplusWatts
					}
				} else {
					allAbove = false
					res = surplus.SurplusWatts
				}
			}
		}
		return res
	}
	// if aleady charging on solar, at least one sample must be above threshold
	res := 0
	for _, surplus := range surpluses {
		if surplus.Timestamp.After(now.Add(-5 * time.Minute)) {
			if state.Charging == ChargeStateChargingOnSolar {
				surplus.SurplusWatts += (state.Amps * 230 * vehicle.NumPhases)
			}
			if surplus.SurplusWatts > res {
				res = surplus.SurplusWatts
			}
		}
	}
	return res
}

func (c *ChargeController) chargeProcessAdjustSolarAmps(vehicle *Vehicle, state *VehicleState, targetAmps int) {
	if state.Charging == ChargeStateChargingOnSolar && targetAmps > 0 && targetAmps != state.Amps {
		// ...and only if the last amps adjustment occured before the latest surplus data came in
		if c.canAdjustSolarAmps(vehicle) {
			err := GetTeslaAPI().Wakeup(vehicle.VIN)
			if err != nil {
				GetDB().LogChargingEvent(vehicle.VIN, LogEventSetChargingAmps, fmt.Sprintf("could not init session with car: %s", err.Error()))
			} else {
				if err := GetTeslaAPI().SetChargeAmps(vehicle.VIN, targetAmps); err != nil {
					GetDB().LogChargingEvent(vehicle.VIN, LogEventSetChargingAmps, "could not set charge amps: "+err.Error())
				} else {
					GetDB().SetVehicleStateAmps(vehicle.VIN, targetAmps)
					GetDB().LogChargingEvent(vehicle.VIN, LogEventSetChargingAmps, fmt.Sprintf("charge amps set to %d", targetAmps))
					SendPushNotification(fmt.Sprintf("Adjusted %s's current to %d amps.", vehicle.DisplayName, targetAmps))
				}
			}
		}
	}
}

func (c *ChargeController) checkChargeProcess(vehicle *Vehicle, state *VehicleState) {
	// if target SoC is reached: stop charging
	if state.SoC >= vehicle.TargetSoC {
		c.stopCharging(vehicle, state)
		return
	}

	// check how the new charging state should be
	targetState, targetAmps := c.checkTargetState(vehicle, state)
	LogDebug(fmt.Sprintf("checkChargeProcess() - target state %d with %d amps for vehicle %s", targetState, targetAmps, vehicle.VIN))

	c.chargeProcessAdjustSolarAmps(vehicle, state, targetAmps)

	// if minimum charge time is not reached, do nothing
	if !c.minimumChargeTimeReached(vehicle, state) {
		LogDebug(fmt.Sprintf("checkChargeProcess() - min charge time not reached for vehicle %s", vehicle.VIN))
		return
	}

	// else, check if charging needs to be stopped
	if targetState == ChargeStateNotCharging {
		c.stopCharging(vehicle, state)
	}
}
