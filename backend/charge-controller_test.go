package main

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
)

func TestChargeControlGetEstimatedChargeDurationMinutes(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		TargetSoC: 70,
		MaxAmps:   16,
		NumPhases: 3,
	}
	s := &VehicleState{
		SoC: 50,
	}
	res := NewChargeController().getEstimatedChargeDurationMinutes(v, s)
	assert.Equal(t, 109, res)
	t.Cleanup(ResetTestDB)
}

func TestChargeControlGetEstimatedChargeDurationMinutesNegative(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		TargetSoC: 70,
		MaxAmps:   16,
		NumPhases: 3,
	}
	s := &VehicleState{
		SoC: 80,
	}
	res := NewChargeController().getEstimatedChargeDurationMinutes(v, s)
	assert.Equal(t, 0, res)
}

func TestChargeControlCheckStartOnSolar(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		MinSurplus:      0,
		SurplusCharging: true,
	}
	s := &VehicleState{
		Amps: 0,
	}
	GetDB().RecordSurplus(v.ID, 4000)
	res, amps := NewChargeController().checkStartOnSolar(v, s)
	assert.True(t, res)
	assert.Equal(t, 5, amps)
}

func TestChargeControlCheckStartOnSolarDisabled(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		MinSurplus:      0,
		SurplusCharging: false,
	}
	s := &VehicleState{
		Amps: 0,
	}
	GetDB().RecordSurplus(v.ID, 4000)
	res, _ := NewChargeController().checkStartOnSolar(v, s)
	assert.False(t, res)
}

func TestChargeControlCheckStartOnSolarNoSurplus(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		MinSurplus:      0,
		SurplusCharging: true,
	}
	s := &VehicleState{
		Amps: 0,
	}
	GetDB().RecordSurplus(v.ID, 0)
	res, _ := NewChargeController().checkStartOnSolar(v, s)
	assert.False(t, res)
}

func TestChargeControlCheckStartOnSolarNotEnoughSurplus(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		MinSurplus:      4000,
		SurplusCharging: true,
	}
	s := &VehicleState{
		Amps: 0,
	}
	GetDB().RecordSurplus(v.ID, 2000)
	res, _ := NewChargeController().checkStartOnSolar(v, s)
	assert.False(t, res)
}

func TestChargeControlCheckStartOnSolarNoRecentSurplus(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		MinSurplus:      0,
		SurplusCharging: true,
	}
	s := &VehicleState{
		Amps: 0,
	}
	GetDB().Connection.Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, datetime('now','-15 minutes'), ?)", v.ID, 4000)
	res, _ := NewChargeController().checkStartOnSolar(v, s)
	assert.False(t, res)
}

func TestChargeControlCheckStartOnSolarMinimalSurplus(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		MinSurplus:      0,
		SurplusCharging: true,
	}
	s := &VehicleState{
		Amps: 0,
	}
	GetDB().RecordSurplus(v.ID, 100)
	res, _ := NewChargeController().checkStartOnSolar(v, s)
	assert.False(t, res)
}

func TestChargeControlCheckStartOnTibber(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		LowcostCharging: true,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyNoDeparturePriceLimit,
		MaxPrice:        20,
	}
	s := &VehicleState{
		SoC: 50,
	}
	now := time.Now().UTC()
	GetDB().SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.15)
	res, amps := NewChargeController().checkStartOnGrid(v, s)
	assert.True(t, res)
	assert.Equal(t, 16, amps)
}

func TestChargeControlCheckStartOnTibberDisabled(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		LowcostCharging: false,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyNoDeparturePriceLimit,
		MaxPrice:        20,
	}
	s := &VehicleState{
		SoC: 50,
	}
	now := time.Now().UTC()
	GetDB().SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.15)
	res, _ := NewChargeController().checkStartOnGrid(v, s)
	assert.False(t, res)
}

func TestChargeControlCheckStartOnTibberNoUpcomingPrices(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		LowcostCharging: true,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyNoDeparturePriceLimit,
		MaxPrice:        20,
	}
	s := &VehicleState{
		SoC: 50,
	}
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	GetDB().SetTibberPrice(v.ID, yesterday.Year(), int(yesterday.Month()), yesterday.Day(), 0, 0.15)
	GetDB().SetTibberPrice(v.ID, yesterday.Year(), int(yesterday.Month()), yesterday.Day(), 1, 0.15)
	GetDB().SetTibberPrice(v.ID, yesterday.Year(), int(yesterday.Month()), yesterday.Day(), 2, 0.15)
	GetDB().SetTibberPrice(v.ID, yesterday.Year(), int(yesterday.Month()), yesterday.Day(), 23, 0.15)
	res, _ := NewChargeController().checkStartOnGrid(v, s)
	assert.False(t, res)
}

func TestChargeControlCheckStartOnTibberMaxPriceExceeded(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		LowcostCharging: true,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyNoDeparturePriceLimit,
		MaxPrice:        20,
	}
	s := &VehicleState{
		SoC: 50,
	}
	now := time.Now().UTC()
	GetDB().SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.3)
	res, _ := NewChargeController().checkStartOnGrid(v, s)
	assert.False(t, res)
}

func TestChargeControlCheckStartOnTibberFutureLowPrices(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		LowcostCharging: true,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyNoDeparturePriceLimit,
		MaxPrice:        20,
	}
	s := &VehicleState{
		SoC: 50,
	}
	now := time.Now().UTC()
	GetDB().SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.3)
	now1 := time.Now().UTC().Add(1 * time.Hour)
	GetDB().SetTibberPrice(v.ID, now1.Year(), int(now1.Month()), now1.Day(), now1.Hour(), 0.15)
	now2 := time.Now().UTC().Add(2 * time.Hour)
	GetDB().SetTibberPrice(v.ID, now2.Year(), int(now2.Month()), now2.Day(), now2.Hour(), 0.18)
	res, _ := NewChargeController().checkStartOnGrid(v, s)
	assert.False(t, res)
}

func TestChargeControlCheckStartOnTibberUpcomingLowerPrices(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		LowcostCharging: true,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyNoDeparturePriceLimit,
		MaxPrice:        20,
	}
	s := &VehicleState{
		SoC: 65,
	}
	now := time.Now().UTC()
	GetDB().SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.15)
	now1 := time.Now().UTC().Add(1 * time.Hour)
	GetDB().SetTibberPrice(v.ID, now1.Year(), int(now1.Month()), now1.Day(), now1.Hour(), 0.10)
	now2 := time.Now().UTC().Add(2 * time.Hour)
	GetDB().SetTibberPrice(v.ID, now2.Year(), int(now2.Month()), now2.Day(), now2.Hour(), 0.12)
	res, _ := NewChargeController().checkStartOnGrid(v, s)
	assert.False(t, res)
}

func TestChargeControlCheckStartOnTibberChargeDuration(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:              123,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		LowcostCharging: true,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyNoDeparturePriceLimit,
		MaxPrice:        20,
	}
	s := &VehicleState{
		SoC: 20,
	}
	now := time.Now().UTC()
	GetDB().SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.15)
	now1 := time.Now().UTC().Add(1 * time.Hour)
	GetDB().SetTibberPrice(v.ID, now1.Year(), int(now1.Month()), now1.Day(), now1.Hour(), 0.10)
	now2 := time.Now().UTC().Add(2 * time.Hour)
	GetDB().SetTibberPrice(v.ID, now2.Year(), int(now2.Month()), now2.Day(), now2.Hour(), 0.12)
	res, amps := NewChargeController().checkStartOnGrid(v, s)
	assert.True(t, res)
	assert.Equal(t, 16, amps)
}

/*
func TestChargeControlCanUpdateVehicleDataNoEventYet(t *testing.T) {
	t.Cleanup(ResetTestDB)
	res := NewChargeController().canUpdateVehicleData(123)
	assert.True(t, res)
}

func TestChargeControlCanUpdateVehicleDataNoUpdatePossible(t *testing.T) {
	t.Cleanup(ResetTestDB)
	GetDB().Connection.Exec("insert into logs values(?, datetime('now','-3 minutes'), ?, ?)", 123, LogEventVehicleUpdateData, "")
	res := NewChargeController().canUpdateVehicleData(123)
	assert.False(t, res)
}

func TestChargeControlCanUpdateVehicleDataUpdatePossible(t *testing.T) {
	t.Cleanup(ResetTestDB)
	GetDB().Connection.Exec("insert into logs values(?, datetime('now','-30 minutes'), ?, ?)", 123, LogEventVehicleUpdateData, "")
	res := NewChargeController().canUpdateVehicleData(123)
	assert.True(t, res)
}
*/

func TestChargeControlMinimumChargeTimeReachedNoEventYet(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:            123,
		MinChargeTime: 15,
	}
	s := &VehicleState{
		Charging: ChargeStateChargingOnSolar,
	}
	res := NewChargeController().minimumChargeTimeReached(v, s)
	assert.True(t, res)
}

func TestChargeControlMinimumChargeTimeReached(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:            123,
		MinChargeTime: 15,
	}
	GetDB().Connection.Exec("insert into logs values(?, datetime('now','-20 minutes'), ?, ?)", v.ID, LogEventChargeStart, "")
	s := &VehicleState{
		Charging: ChargeStateChargingOnSolar,
	}
	res := NewChargeController().minimumChargeTimeReached(v, s)
	assert.True(t, res)
}

func TestChargeControl_MinimumChargeTimeNotReached(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:            123,
		MinChargeTime: 15,
	}
	GetDB().Connection.Exec("insert into logs values(?, datetime('now','-10 minutes'), ?, ?)", v.ID, LogEventChargeStart, "")
	s := &VehicleState{
		Charging: ChargeStateChargingOnSolar,
	}
	res := NewChargeController().minimumChargeTimeReached(v, s)
	assert.False(t, res)
}

func TestChargeControl_getNextDeparture_NextDay(t *testing.T) {
	v := &Vehicle{
		ID:              123,
		LowcostCharging: true,
		GridStrategy:    GridStrategyDepartureNoPriceLimit,
		DepartDays:      "235",
		DepartTime:      "07:30:00",
	}
	cc := NewTestChargeController()
	GlobalMockTime.CurTime = GetNextMondayMidnight()
	is, _ := cc.getNextDeparture(v)
	should := GetNextMondayMidnight().AddDate(0, 0, 1)
	should = time.Date(should.Year(), should.Month(), should.Day(), 7, 30, 0, 0, should.Location())
	assert.Equal(t, &should, is)
}

func TestChargeControl_getNextDeparture_SameDay(t *testing.T) {
	v := &Vehicle{
		ID:              123,
		LowcostCharging: true,
		GridStrategy:    GridStrategyDepartureNoPriceLimit,
		DepartDays:      "235",
		DepartTime:      "07:30:00",
	}
	cc := NewTestChargeController()
	GlobalMockTime.CurTime = GetNextMondayMidnight()
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.AddDate(0, 0, 2)
	is, _ := cc.getNextDeparture(v)
	should := GetNextMondayMidnight().AddDate(0, 0, 2)
	should = time.Date(should.Year(), should.Month(), should.Day(), 7, 30, 0, 0, should.Location())
	assert.Equal(t, &should, is)
}

func TestChargeControl_getNextDeparture_NextDayDueToTime(t *testing.T) {
	v := &Vehicle{
		ID:              123,
		LowcostCharging: true,
		GridStrategy:    GridStrategyDepartureNoPriceLimit,
		DepartDays:      "235",
		DepartTime:      "07:30:00",
	}
	cc := NewTestChargeController()
	GlobalMockTime.CurTime = GetNextMondayMidnight()
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.AddDate(0, 0, 1)
	GlobalMockTime.CurTime = time.Date(GlobalMockTime.CurTime.Year(), GlobalMockTime.CurTime.Month(), GlobalMockTime.CurTime.Day(), 8, 0, 0, 0, GlobalMockTime.CurTime.Location())
	is, _ := cc.getNextDeparture(v)
	should := GetNextMondayMidnight().AddDate(0, 0, 2)
	should = time.Date(should.Year(), should.Month(), should.Day(), 7, 30, 0, 0, should.Location())
	assert.Equal(t, &should, is)
}

func TestChargeControl_getNextDeparture_NextWeek(t *testing.T) {
	v := &Vehicle{
		ID:              123,
		LowcostCharging: true,
		GridStrategy:    GridStrategyDepartureNoPriceLimit,
		DepartDays:      "235",
		DepartTime:      "07:30:00",
	}
	cc := NewTestChargeController()
	GlobalMockTime.CurTime = GetNextMondayMidnight()
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.AddDate(0, 0, 5)
	is, _ := cc.getNextDeparture(v)
	should := GetNextMondayMidnight().AddDate(0, 0, 8)
	should = time.Date(should.Year(), should.Month(), should.Day(), 7, 30, 0, 0, should.Location())
	assert.Equal(t, &should, is)
}

func TestChargeControl_SolarCharging(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{
		ID:              123,
		VIN:             "123",
		UserID:          "999",
		Enabled:         true,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		SurplusCharging: true,
		MinSurplus:      2000,
		MinChargeTime:   15,
		LowcostCharging: false,
	}
	GetDB().CreateUpdateVehicle(v)
	GetDB().SetVehicleStateSoC(v.ID, 50)
	GetDB().SetVehicleStatePluggedIn(v.ID, true)
	GetDB().SetVehicleStateCharging(v.ID, ChargeStateNotCharging)
	cc := NewTestChargeController()

	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("GetOrRefreshAccessToken", v.UserID).Return("token")
	api.On("InitSession", mock.Anything, mock.Anything).Return(&vehicle.Vehicle{}, nil)
	api.On("SetChargeLimit", mock.Anything, mock.Anything).Return(nil)
	api.On("SetChargeAmps", mock.Anything, mock.Anything).Return(nil)
	api.On("ChargeStart", mock.Anything).Return(nil)
	api.On("ChargeStop", mock.Anything).Return(nil)
	api.On("Wakeup", mock.Anything).Return(nil)
	vData := &TeslaAPIVehicleData{
		VehicleID: 123,
		ChargeState: TeslaAPIChargeState{
			BatteryLevel: 53,
		},
	}
	api.On("GetVehicleData", mock.Anything).Return(vData, nil)
	UpdateTeslaAPIMockData(api, 123, 53, "")

	// on start, no surplus records, so vehicle is not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour).Add(-1 * time.Duration(GlobalMockTime.CurTime.Minute()) * time.Minute)
	cc.OnTick()
	state := GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "")

	// record a surplus too low, still no charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(5 * time.Minute) // +5
	GetDB().RecordSurplus(v.ID, 500)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "Charging")

	// record a surplus large enough, should start charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(5 * time.Minute) // +10
	GetDB().RecordSurplus(v.ID, 2500)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnSolar, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "Charging")

	// record a surplus not large enough anymore, but should keep on charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(10 * time.Minute) // +20
	GetDB().RecordSurplus(v.ID, -500)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnSolar, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "")

	// charging should end now
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(10 * time.Minute) // +30
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
}

func TestChargeControl_SolarCharging_AmpsAdjustment(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{
		ID:              123,
		VIN:             "123",
		UserID:          "999",
		Enabled:         true,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		SurplusCharging: true,
		MinSurplus:      2000,
		MinChargeTime:   15,
		LowcostCharging: false,
	}
	GetDB().CreateUpdateVehicle(v)
	GetDB().SetVehicleStateSoC(v.ID, 50)
	GetDB().SetVehicleStatePluggedIn(v.ID, true)
	GetDB().SetVehicleStateCharging(v.ID, ChargeStateNotCharging)
	cc := NewTestChargeController()

	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("GetOrRefreshAccessToken", v.UserID).Return("token")
	api.On("InitSession", mock.Anything, mock.Anything).Return(&vehicle.Vehicle{}, nil)
	api.On("SetChargeLimit", mock.Anything, mock.Anything).Return(nil)
	api.On("SetChargeAmps", mock.Anything, mock.Anything).Return(nil)
	api.On("ChargeStart", mock.Anything).Return(nil)
	api.On("ChargeStop", mock.Anything).Return(nil)
	api.On("Wakeup", mock.Anything).Return(nil)
	vData := &TeslaAPIVehicleData{
		VehicleID: 123,
		ChargeState: TeslaAPIChargeState{
			BatteryLevel: 53,
		},
	}
	api.On("GetVehicleData", "token", mock.Anything).Return(vData, nil)
	UpdateTeslaAPIMockData(api, 123, 53, "")

	// on start, no surplus records, so vehicle is not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour).Add(-1 * time.Duration(GlobalMockTime.CurTime.Minute()) * time.Minute)
	cc.OnTick()
	state := GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "")

	// record a surplus too low, still no charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(5 * time.Minute) // +5
	GetDB().RecordSurplus(v.ID, 500)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "Charging")

	// record a surplus large enough, should start charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(5 * time.Minute) // +10
	GetDB().RecordSurplus(v.ID, 2500)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnSolar, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "Charging")

	// record a surplus not large enough anymore, but should keep on charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(10 * time.Minute) // +20
	GetDB().RecordSurplus(v.ID, -500)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnSolar, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "")

	// charging should end now
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(10 * time.Minute) // +30
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
}

func TestChargeControl_TibberChargingNoDeparturePriceLimit(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{
		ID:              123,
		VIN:             "123",
		UserID:          "999",
		Enabled:         true,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		SurplusCharging: false,
		LowcostCharging: true,
		MaxPrice:        20,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyNoDeparturePriceLimit,
	}
	GetDB().CreateUpdateVehicle(v)
	GetDB().SetVehicleStateSoC(v.ID, 50)
	GetDB().SetVehicleStatePluggedIn(v.ID, true)
	GetDB().SetVehicleStateCharging(v.ID, ChargeStateNotCharging)
	cc := NewTestChargeController()

	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("GetOrRefreshAccessToken", v.UserID).Return("token")
	api.On("InitSession", mock.Anything, mock.Anything).Return(&vehicle.Vehicle{}, nil)
	api.On("SetChargeLimit", mock.Anything, mock.Anything).Return(nil)
	api.On("SetChargeAmps", mock.Anything, mock.Anything).Return(nil)
	api.On("ChargeStart", mock.Anything).Return(nil)
	api.On("ChargeStop", mock.Anything).Return(nil)
	api.On("Wakeup", mock.Anything).Return(nil)
	UpdateTeslaAPIMockData(api, 123, 53, "")

	now := time.Now().UTC()
	SetTibberTestPrice(v.ID, now.Add(time.Hour*1), 0.25) // 0
	SetTibberTestPrice(v.ID, now.Add(time.Hour*2), 0.27) // 1
	SetTibberTestPrice(v.ID, now.Add(time.Hour*3), 0.19) // 2
	SetTibberTestPrice(v.ID, now.Add(time.Hour*4), 0.15) // 3
	SetTibberTestPrice(v.ID, now.Add(time.Hour*5), 0.18) // 4
	SetTibberTestPrice(v.ID, now.Add(time.Hour*6), 0.30) // 5

	// on start, price is above maximum, vehicle is not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour).Add(-1 * time.Duration(GlobalMockTime.CurTime.Minute()) * time.Minute)
	cc.OnTick()
	state := GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "")

	// +1 hour, price still above maximum
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour) // +1
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "")

	// +2 hours, price is below max, but highest among below-threshold prices, so still no charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour) // +1
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "Charging")

	// +3 hours, price is below max and even though not minimum, this hour is required to reach the desired SoC
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour) // +1
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "Charging")

	// +4 hours, price is minimal, still charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour) // +1
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 53, "")

	// +5 hours, charging should stop
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour) // +1
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
}

func TestChargeControl_TibberChargingDepartureNoPriceLimit(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{
		ID:              123,
		VIN:             "123",
		UserID:          "999",
		Enabled:         true,
		TargetSoC:       80,
		MaxAmps:         16,
		NumPhases:       3,
		SurplusCharging: false,
		LowcostCharging: true,
		MaxPrice:        20,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyDepartureNoPriceLimit,
		DepartDays:      "135",
		DepartTime:      "07:00:00",
	}
	GetDB().CreateUpdateVehicle(v)
	GetDB().SetVehicleStateSoC(v.ID, 40)
	GetDB().SetVehicleStatePluggedIn(v.ID, true)
	GetDB().SetVehicleStateCharging(v.ID, ChargeStateNotCharging)
	cc := NewTestChargeController()

	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("GetOrRefreshAccessToken", v.UserID).Return("token")
	api.On("InitSession", mock.Anything, mock.Anything).Return(&vehicle.Vehicle{}, nil)
	api.On("SetChargeLimit", mock.Anything, mock.Anything).Return(nil)
	api.On("SetChargeAmps", mock.Anything, mock.Anything).Return(nil)
	api.On("ChargeStart", mock.Anything).Return(nil)
	api.On("ChargeStop", mock.Anything).Return(nil)
	api.On("Wakeup", mock.Anything).Return(nil)
	UpdateTeslaAPIMockData(api, 123, 40, "")

	now := GetNextMondayMidnight()

	SetTibberTestPrice(v.ID, now.Add(time.Hour*0), 0.32)  // 00:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*1), 0.25)  // 01:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*2), 0.27)  // 02:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*3), 0.19)  // 03:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*4), 0.15)  // 04:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*5), 0.18)  // 05:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*6), 0.30)  // 06:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*7), 0.15)  // 07:00 <-- departure
	SetTibberTestPrice(v.ID, now.Add(time.Hour*8), 0.08)  // 08:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*9), 0.07)  // 09:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*10), 0.15) // 10:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*11), 0.50) // 11:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*12), 0.10) // 12:00

	// Calculated charge duration 40 -> 80: 3.7 hours

	// 00:00 - not charging
	GlobalMockTime.CurTime = now
	cc.OnTick()
	state := GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 40, "")

	// 00:30 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 40, "Charging")

	// 01:00 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 45, "Charging")

	// 01:30 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 51, "Charging")

	// 02:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 51, "Charging")

	// 02:30 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 51, "Charging")

	// 03:00 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 56, "Charging")

	// 03:30 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 62, "Charging")

	// 04:00 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 68, "Charging")

	// 04:30 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 73, "Charging")

	// 05:00 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 77, "Charging")

	// 05:30 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 80, "")

	// 06:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 80, "")

	// 06:30 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 80, "")

	// 07:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 80, "")

	// 07:30 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 80, "")

	// 08:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 80, "")
}

func TestChargeControl_TibberChargingDepartureNoPriceLimit2(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{
		ID:              123,
		VIN:             "123",
		UserID:          "999",
		Enabled:         true,
		TargetSoC:       75,
		MaxAmps:         16,
		NumPhases:       3,
		SurplusCharging: false,
		LowcostCharging: true,
		MaxPrice:        20,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyDepartureNoPriceLimit,
		DepartDays:      "123",
		DepartTime:      "07:00:00",
	}
	GetDB().CreateUpdateVehicle(v)
	GetDB().SetVehicleStateSoC(v.ID, 63)
	GetDB().SetVehicleStatePluggedIn(v.ID, true)
	GetDB().SetVehicleStateCharging(v.ID, ChargeStateNotCharging)
	cc := NewTestChargeController()

	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("GetOrRefreshAccessToken", v.UserID).Return("token")
	api.On("InitSession", mock.Anything, mock.Anything).Return(&vehicle.Vehicle{}, nil)
	api.On("SetChargeLimit", mock.Anything, mock.Anything).Return(nil)
	api.On("SetChargeAmps", mock.Anything, mock.Anything).Return(nil)
	api.On("ChargeStart", mock.Anything).Return(nil)
	api.On("ChargeStop", mock.Anything).Return(nil)
	api.On("Wakeup", mock.Anything).Return(nil)
	UpdateTeslaAPIMockData(api, 123, 63, "")

	now := GetNextMondayMidnight()
	now = now.Add(time.Hour * 22)

	SetTibberTestPrice(v.ID, now.Add(time.Hour*0), 0.231900006532669)  // 22:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*1), 0.23989999294281)   // 23:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*2), 0.236100003123283)  // 00:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*3), 0.233700007200241)  // 01:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*4), 0.23029999434948)   // 02:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*5), 0.229800000786781)  // 03:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*6), 0.233999997377396)  // 04:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*7), 0.242400005459785)  // 05:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*8), 0.256099998950958)  // 06:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*9), 0.264299988746643)  // 07:00 <-- departure
	SetTibberTestPrice(v.ID, now.Add(time.Hour*10), 0.24770000576973)  // 08:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*11), 0.240600004792213) // 09:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*12), 0.234300002455711) // 10:00

	state := GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)

	// Calculated charge duration 63 -> 80: 3.7 hours

	// 22:00 - charging
	GlobalMockTime.CurTime = now
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 63, "")

	// 22:30 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 63, "")

	// 23:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	log.Println(GlobalMockTime.CurTime)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 63, "")

	// 23:30 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 63, "")

	// 00:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 63, "")

	// 00:30 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 63, "")

	// 01:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 63, "")

	// 01:30 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 63, "Charging")

	// 02:00 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 65, "Charging")

	// 02:30 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 71, "Charging")

	// 03:00 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 75, "")

	// 03:30 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 75, "")

	// 04:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Minute * 30)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 75, "")

}

func TestChargeControl_TibberChargingDepartureWithPriceLimit(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{
		ID:              123,
		VIN:             "123",
		UserID:          "999",
		Enabled:         true,
		TargetSoC:       80,
		MaxAmps:         16,
		NumPhases:       3,
		SurplusCharging: false,
		LowcostCharging: true,
		MaxPrice:        20,
		GridProvider:    GridProviderTibber,
		GridStrategy:    GridStrategyDepartureWithPriceLimit,
		DepartDays:      "135",
		DepartTime:      "07:00:00",
	}
	GetDB().CreateUpdateVehicle(v)
	GetDB().SetVehicleStateSoC(v.ID, 40)
	GetDB().SetVehicleStatePluggedIn(v.ID, true)
	GetDB().SetVehicleStateCharging(v.ID, ChargeStateNotCharging)
	cc := NewTestChargeController()

	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("GetOrRefreshAccessToken", v.UserID).Return("token")
	api.On("InitSession", mock.Anything, mock.Anything).Return(&vehicle.Vehicle{}, nil)
	api.On("SetChargeLimit", mock.Anything, mock.Anything).Return(nil)
	api.On("SetChargeAmps", mock.Anything, mock.Anything).Return(nil)
	api.On("ChargeStart", mock.Anything).Return(nil)
	api.On("ChargeStop", mock.Anything).Return(nil)
	api.On("Wakeup", mock.Anything).Return(nil)
	UpdateTeslaAPIMockData(api, 123, 40, "")

	now := GetNextMondayMidnight()

	SetTibberTestPrice(v.ID, now.Add(time.Hour*0), 0.32)  // 00:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*1), 0.25)  // 01:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*2), 0.27)  // 02:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*3), 0.20)  // 03:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*4), 0.15)  // 04:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*5), 0.18)  // 05:00 <-- charge
	SetTibberTestPrice(v.ID, now.Add(time.Hour*6), 0.30)  // 06:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*7), 0.15)  // 07:00 <-- departure
	SetTibberTestPrice(v.ID, now.Add(time.Hour*8), 0.08)  // 08:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*9), 0.07)  // 09:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*10), 0.15) // 10:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*11), 0.50) // 11:00
	SetTibberTestPrice(v.ID, now.Add(time.Hour*12), 0.10) // 12:00

	// Calculated charge duration 40 -> 80: 3.7 hours

	// 00:00 - not charging
	GlobalMockTime.CurTime = now
	cc.OnTick()
	state := GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 40, "")

	// 01:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Hour * 1)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 40, "")

	// 02:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Hour * 1)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 40, "Charging")

	// 03:00 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Hour * 1)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 51, "Charging")

	// 04:00 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Hour * 1)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 62, "Charging")

	// 05:00 - charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Hour * 1)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 73, "")

	// 06:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Hour * 1)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 73, "")

	// 07:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Hour * 1)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 73, "")

	// 08:00 - not charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Hour * 1)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	UpdateTeslaAPIMockData(api, 123, 73, "")
}

func TestChargeControl_containsPricesUntilDeparture_true(t *testing.T) {
	t.Cleanup(ResetTestDB)

	now := time.Now().UTC()
	departure := time.Date(2023, 2, 11, 15, 0, 0, 0, now.Location())
	prices := []*GridPrice{
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 13, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 14, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 15, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 16, 0, 0, 0, time.UTC)},
	}

	cc := NewTestChargeController()
	assert.True(t, cc.containsPricesUntilDeparture(prices, departure))
}

func TestChargeControl_containsPricesUntilDeparture_false(t *testing.T) {
	t.Cleanup(ResetTestDB)

	now := time.Now().UTC()
	departure := time.Date(2023, 2, 11, 18, 0, 0, 0, now.Location())
	prices := []*GridPrice{
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 13, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 14, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 15, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 16, 0, 0, 0, time.UTC)},
	}

	cc := NewTestChargeController()
	assert.False(t, cc.containsPricesUntilDeparture(prices, departure))
}

func TestChargeControl_containsPricesUntilDeparture_edge(t *testing.T) {
	t.Cleanup(ResetTestDB)

	now := time.Now().UTC()
	departure := time.Date(2023, 2, 11, 16, 0, 0, 0, now.Location())
	prices := []*GridPrice{
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 13, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 14, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 15, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 16, 0, 0, 0, time.UTC)},
	}

	cc := NewTestChargeController()
	assert.True(t, cc.containsPricesUntilDeparture(prices, departure))
}

func TestChargeControl_containsPricesUntilDeparture_edge2(t *testing.T) {
	t.Cleanup(ResetTestDB)

	now := time.Now().UTC()
	departure := time.Date(2023, 2, 11, 17, 0, 0, 0, now.Location())
	prices := []*GridPrice{
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 13, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 14, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 15, 0, 0, 0, time.UTC)},
		{Total: 0.3, StartsAt: time.Date(2023, 2, 11, 16, 0, 0, 0, time.UTC)},
	}

	cc := NewTestChargeController()
	assert.True(t, cc.containsPricesUntilDeparture(prices, departure))
}

func TestChargeControl_canAdjustSolarAmps_yes(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{ID: 123}
	GetDB().GetConnection().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow()), 2000)
	GetDB().GetConnection().Exec("insert into logs values(?, ?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow().Add(-10*time.Minute)), LogEventSetChargingAmps, "")
	cc := NewTestChargeController()
	assert.True(t, cc.canAdjustSolarAmps(v))
}

func TestChargeControl_canAdjustSolarAmps_yesEdge(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{ID: 123}
	GetDB().GetConnection().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow()), 2000)
	GetDB().GetConnection().Exec("insert into logs values(?, ?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow().Add(-5*time.Minute)), LogEventSetChargingAmps, "")
	cc := NewTestChargeController()
	assert.True(t, cc.canAdjustSolarAmps(v))
}

func TestChargeControl_canAdjustSolarAmps_no(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{ID: 123}
	GetDB().GetConnection().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow()), 2000)
	GetDB().GetConnection().Exec("insert into logs values(?, ?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow().Add(-4*time.Minute)), LogEventSetChargingAmps, "")
	cc := NewTestChargeController()
	assert.False(t, cc.canAdjustSolarAmps(v))
}

func TestChargeControl_getActualSurplus_charging(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{ID: 123, NumPhases: 3}
	s := &VehicleState{Charging: ChargeStateChargingOnSolar, Amps: 5}
	GetDB().RecordSurplus(v.ID, -2000)
	cc := NewTestChargeController()
	surplus := cc.getActualSurplus(v, s)
	assert.NotNil(t, surplus)
	assert.Equal(t, 1450, surplus)
}

func TestChargeControl_getActualSurplus_charging2(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{ID: 123, NumPhases: 1}
	s := &VehicleState{Charging: ChargeStateChargingOnSolar, Amps: 1}
	GetDB().RecordSurplus(v.ID, 0)
	cc := NewTestChargeController()
	surplus := cc.getActualSurplus(v, s)
	assert.NotNil(t, surplus)
	assert.Equal(t, 230, surplus)
}

func TestChargeControl_getActualSurplus_notCharging(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{ID: 123, NumPhases: 3}
	s := &VehicleState{Charging: ChargeStateNotCharging, Amps: 0}
	GetDB().RecordSurplus(v.ID, 2000)
	cc := NewTestChargeController()
	surplus := cc.getActualSurplus(v, s)
	assert.NotNil(t, surplus)
	assert.Equal(t, 2000, surplus)
}

func TestChargeControl_getActualSurplus_multipleRecords(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{ID: 123, NumPhases: 3}
	s := &VehicleState{Charging: ChargeStateNotCharging, Amps: 0}
	GetDB().GetConnection().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow().Add(-2*time.Minute)), 1000)
	GetDB().GetConnection().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow().Add(-1*time.Minute)), 2000)
	GetDB().GetConnection().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow().Add(-0*time.Minute)), 3000)
	cc := NewTestChargeController()
	surplus := cc.getActualSurplus(v, s)
	assert.NotNil(t, surplus)
	assert.Equal(t, 2500, surplus)
}

func TestChargeControl_getActualSurplus_oldRecords(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{ID: 123, NumPhases: 3}
	s := &VehicleState{Charging: ChargeStateNotCharging, Amps: 0}
	GetDB().GetConnection().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow().Add(-20*time.Minute)), 1000)
	GetDB().GetConnection().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow().Add(-10*time.Minute)), 2000)
	GetDB().GetConnection().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, ?, ?)", v.ID, GetDB().formatSqliteDatetime(GetDB().Time.UTCNow().Add(-1*time.Minute)), 3000)
	cc := NewTestChargeController()
	surplus := cc.getActualSurplus(v, s)
	assert.NotNil(t, surplus)
	assert.Equal(t, 3000, surplus)
}
