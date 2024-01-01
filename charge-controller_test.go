package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	GetDB().RecordSurplus(v.ID, 4000)
	res, amps := NewChargeController().checkStartOnSolar(v)
	assert.True(t, res)
	assert.Equal(t, 6, amps)
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
	GetDB().RecordSurplus(v.ID, 4000)
	res, _ := NewChargeController().checkStartOnSolar(v)
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
	GetDB().RecordSurplus(v.ID, 0)
	res, _ := NewChargeController().checkStartOnSolar(v)
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
	GetDB().RecordSurplus(v.ID, 2000)
	res, _ := NewChargeController().checkStartOnSolar(v)
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
	GetDB().Connection.Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, datetime('now','-15 minutes'), ?)", v.ID, 4000)
	res, _ := NewChargeController().checkStartOnSolar(v)
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
	GetDB().RecordSurplus(v.ID, 100)
	res, _ := NewChargeController().checkStartOnSolar(v)
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
		MaxPrice:        20,
	}
	s := &VehicleState{
		SoC: 50,
	}
	now := time.Now().UTC()
	GetDB().SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.15)
	res, amps := NewChargeController().checkStartOnTibber(v, s)
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
		MaxPrice:        20,
	}
	s := &VehicleState{
		SoC: 50,
	}
	now := time.Now().UTC()
	GetDB().SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.15)
	res, _ := NewChargeController().checkStartOnTibber(v, s)
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
	res, _ := NewChargeController().checkStartOnTibber(v, s)
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
		MaxPrice:        20,
	}
	s := &VehicleState{
		SoC: 50,
	}
	now := time.Now().UTC()
	GetDB().SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.3)
	res, _ := NewChargeController().checkStartOnTibber(v, s)
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
	res, _ := NewChargeController().checkStartOnTibber(v, s)
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
	res, _ := NewChargeController().checkStartOnTibber(v, s)
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
	res, amps := NewChargeController().checkStartOnTibber(v, s)
	assert.True(t, res)
	assert.Equal(t, 16, amps)
}

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

func TestChargeControlMinimumChargeTimeReachedNoEventYet(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:            123,
		MinChargeTime: 15,
	}
	res := NewChargeController().minimumChargeTimeReached(v)
	assert.True(t, res)
}

func TestChargeControlMinimumChargeTimeReached(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:            123,
		MinChargeTime: 15,
	}
	GetDB().Connection.Exec("insert into logs values(?, datetime('now','-20 minutes'), ?, ?)", v.ID, LogEventChargeStart, "")
	res := NewChargeController().minimumChargeTimeReached(v)
	assert.True(t, res)
}

func TestChargeControl_MinimumChargeTimeNotReached(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:            123,
		MinChargeTime: 15,
	}
	GetDB().Connection.Exec("insert into logs values(?, datetime('now','-10 minutes'), ?, ?)", v.ID, LogEventChargeStart, "")
	res := NewChargeController().minimumChargeTimeReached(v)
	assert.False(t, res)
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
	api.On("WakeUpVehicle", "token", mock.Anything).Return(nil)
	api.On("SetChargeLimit", "token", mock.Anything, mock.Anything).Return(true, nil)
	api.On("SetChargeAmps", "token", mock.Anything, mock.Anything).Return(true, nil)
	api.On("ChargeStart", "token", mock.Anything).Return(true, nil)
	api.On("ChargeStop", "token", mock.Anything).Return(true, nil)
	api.On("SetScheduledCharging", "token", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	vData := &TeslaAPIVehicleData{
		VehicleID: 123,
		ChargeState: TeslaAPIChargeState{
			BatteryLevel: 53,
		},
	}
	api.On("GetVehicleData", "token", mock.Anything).Return(vData, nil)

	// on start, no surplus records, so vehicle is not charging
	cc.OnTick()
	state := GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)

	// record a surplus too low, still no charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Minute) // +1
	GetDB().RecordSurplus(v.ID, 500)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)

	// record a surplus large enough, should start charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Minute) // +2
	GetDB().RecordSurplus(v.ID, 2500)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnSolar, state.Charging)

	// record a surplus not large enough anymore, but should keep on charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(10 * time.Minute) // +12
	GetDB().RecordSurplus(v.ID, 500)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnSolar, state.Charging)

	// charging should end now
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(time.Duration(10+5-int(GlobalMockTime.CurTime.Minute()%5)) * time.Minute)
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
}

func TestChargeControl_TibberCharging(t *testing.T) {
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
	}
	GetDB().CreateUpdateVehicle(v)
	GetDB().SetVehicleStateSoC(v.ID, 50)
	GetDB().SetVehicleStatePluggedIn(v.ID, true)
	GetDB().SetVehicleStateCharging(v.ID, ChargeStateNotCharging)
	cc := NewTestChargeController()

	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("GetOrRefreshAccessToken", v.UserID).Return("token")
	api.On("WakeUpVehicle", "token", mock.Anything).Return(nil)
	api.On("SetChargeLimit", "token", mock.Anything, mock.Anything).Return(true, nil)
	api.On("SetChargeAmps", "token", mock.Anything, mock.Anything).Return(true, nil)
	api.On("ChargeStart", "token", mock.Anything).Return(true, nil)
	api.On("ChargeStop", "token", mock.Anything).Return(true, nil)
	api.On("SetScheduledCharging", "token", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	vData := &TeslaAPIVehicleData{
		VehicleID: 123,
		ChargeState: TeslaAPIChargeState{
			BatteryLevel: 53,
		},
	}
	api.On("GetVehicleData", "token", mock.Anything).Return(vData, nil)

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

	// +1 hour, price still above maximum
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour) // +1
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)

	// +2 hours, price is below max, but highest among below-threshold prices, so still no charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour) // +1
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)

	// +3 hours, price is below max and even though not minimum, this hour is required to reach the desired SoC
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour) // +1
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)

	// +4 hours, price is minimal, still charging
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour) // +1
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)

	// +5 hours, charging should stop
	GlobalMockTime.CurTime = GlobalMockTime.CurTime.Add(1 * time.Hour) // +1
	cc.OnTick()
	state = GetDB().GetVehicleState(v.ID)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
}
