package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	res := ChargeControlGetEstimatedChargeDurationMinutes(v, s)
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
	res := ChargeControlGetEstimatedChargeDurationMinutes(v, s)
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
	RecordSurplus(v.ID, 4000)
	res, amps := ChargeControlCheckStartOnSolar(v)
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
	RecordSurplus(v.ID, 4000)
	res, _ := ChargeControlCheckStartOnSolar(v)
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
	RecordSurplus(v.ID, 0)
	res, _ := ChargeControlCheckStartOnSolar(v)
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
	RecordSurplus(v.ID, 2000)
	res, _ := ChargeControlCheckStartOnSolar(v)
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
	GetDB().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, datetime('now','-15 minutes'), ?)", v.ID, 4000)
	res, _ := ChargeControlCheckStartOnSolar(v)
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
	RecordSurplus(v.ID, 100)
	res, _ := ChargeControlCheckStartOnSolar(v)
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
	SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.15)
	res, amps := ChargeControlCheckStartOnTibber(v, s)
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
	SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.15)
	res, _ := ChargeControlCheckStartOnTibber(v, s)
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
	SetTibberPrice(v.ID, yesterday.Year(), int(yesterday.Month()), yesterday.Day(), 0, 0.15)
	SetTibberPrice(v.ID, yesterday.Year(), int(yesterday.Month()), yesterday.Day(), 1, 0.15)
	SetTibberPrice(v.ID, yesterday.Year(), int(yesterday.Month()), yesterday.Day(), 2, 0.15)
	SetTibberPrice(v.ID, yesterday.Year(), int(yesterday.Month()), yesterday.Day(), 23, 0.15)
	res, _ := ChargeControlCheckStartOnTibber(v, s)
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
	SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.3)
	res, _ := ChargeControlCheckStartOnTibber(v, s)
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
	SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.3)
	now1 := time.Now().UTC().Add(1 * time.Hour)
	SetTibberPrice(v.ID, now1.Year(), int(now1.Month()), now1.Day(), now1.Hour(), 0.15)
	now2 := time.Now().UTC().Add(2 * time.Hour)
	SetTibberPrice(v.ID, now2.Year(), int(now2.Month()), now2.Day(), now2.Hour(), 0.18)
	res, _ := ChargeControlCheckStartOnTibber(v, s)
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
	SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.15)
	now1 := time.Now().UTC().Add(1 * time.Hour)
	SetTibberPrice(v.ID, now1.Year(), int(now1.Month()), now1.Day(), now1.Hour(), 0.10)
	now2 := time.Now().UTC().Add(2 * time.Hour)
	SetTibberPrice(v.ID, now2.Year(), int(now2.Month()), now2.Day(), now2.Hour(), 0.12)
	res, _ := ChargeControlCheckStartOnTibber(v, s)
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
	SetTibberPrice(v.ID, now.Year(), int(now.Month()), now.Day(), now.Hour(), 0.15)
	now1 := time.Now().UTC().Add(1 * time.Hour)
	SetTibberPrice(v.ID, now1.Year(), int(now1.Month()), now1.Day(), now1.Hour(), 0.10)
	now2 := time.Now().UTC().Add(2 * time.Hour)
	SetTibberPrice(v.ID, now2.Year(), int(now2.Month()), now2.Day(), now2.Hour(), 0.12)
	res, amps := ChargeControlCheckStartOnTibber(v, s)
	assert.True(t, res)
	assert.Equal(t, 16, amps)
}

func TestChargeControlCanUpdateVehicleDataNoEventYet(t *testing.T) {
	t.Cleanup(ResetTestDB)
	res := ChargeControlCanUpdateVehicleData(123)
	assert.True(t, res)
}

func TestChargeControlCanUpdateVehicleDataNoUpdatePossible(t *testing.T) {
	t.Cleanup(ResetTestDB)
	GetDB().Exec("insert into logs values(?, datetime('now','-3 minutes'), ?, ?)", 123, LogEventVehicleUpdateData, "")
	res := ChargeControlCanUpdateVehicleData(123)
	assert.False(t, res)
}

func TestChargeControlCanUpdateVehicleDataUpdatePossible(t *testing.T) {
	t.Cleanup(ResetTestDB)
	GetDB().Exec("insert into logs values(?, datetime('now','-30 minutes'), ?, ?)", 123, LogEventVehicleUpdateData, "")
	res := ChargeControlCanUpdateVehicleData(123)
	assert.True(t, res)
}

func TestChargeControlMinimumChargeTimeReachedNoEventYet(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:            123,
		MinChargeTime: 15,
	}
	res := ChargeControlMinimumChargeTimeReached(v)
	assert.True(t, res)
}

func TestChargeControlMinimumChargeTimeReached(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:            123,
		MinChargeTime: 15,
	}
	GetDB().Exec("insert into logs values(?, datetime('now','-20 minutes'), ?, ?)", v.ID, LogEventChargeStart, "")
	res := ChargeControlMinimumChargeTimeReached(v)
	assert.True(t, res)
}

func TestChargeControlMinimumChargeTimeNotReached(t *testing.T) {
	t.Cleanup(ResetTestDB)
	v := &Vehicle{
		ID:            123,
		MinChargeTime: 15,
	}
	GetDB().Exec("insert into logs values(?, datetime('now','-10 minutes'), ?, ?)", v.ID, LogEventChargeStart, "")
	res := ChargeControlMinimumChargeTimeReached(v)
	assert.False(t, res)
}
