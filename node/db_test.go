package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDB_SetVehicleStates(t *testing.T) {
	t.Cleanup(ResetTestDB)

	GetDB().SetVehicleStateCharging("123", ChargeStateChargingOnSolar)
	GetDB().SetVehicleStateAmps("123", 5)
	GetDB().SetVehicleStatePluggedIn("123", true)
	GetDB().SetVehicleStateSoC("123", 55)

	state := GetDB().GetVehicleState("123")
	assert.NotNil(t, state)
	assert.Equal(t, ChargeStateChargingOnSolar, state.Charging)
	assert.Equal(t, 5, state.Amps)
	assert.Equal(t, 55, state.SoC)
	assert.Equal(t, true, state.PluggedIn)

}

func TestDB_CRUDVehicle(t *testing.T) {
	t.Cleanup(ResetTestDB)

	vehicle := &Vehicle{
		VIN:             "789",
		DisplayName:     "Model S",
		Enabled:         true,
		TargetSoC:       65,
		MaxAmps:         8,
		NumPhases:       3,
		SurplusCharging: true,
		MinSurplus:      1250,
		MinChargeTime:   25,
		LowcostCharging: true,
		MaxPrice:        22,
		GridProvider:    "tibber",
		GridStrategy:    2,
		DepartDays:      "1357",
		DepartTime:      "07:15:00",
		TibberToken:     "def",
	}
	GetDB().CreateUpdateVehicle(vehicle)

	vehicle2 := GetDB().GetVehicleByVIN(vehicle.VIN)
	assert.NotNil(t, vehicle2)
	assert.EqualValues(t, vehicle, vehicle2)

	GetDB().DeleteVehicle(vehicle.VIN)
	vehicle2 = GetDB().GetVehicleByVIN(vehicle.VIN)
	assert.Nil(t, vehicle2)
}

func TestDB_GetVehicles(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v1 := &Vehicle{VIN: "1", DisplayName: "V 1"}
	v2 := &Vehicle{VIN: "2", DisplayName: "V 2"}
	v3 := &Vehicle{VIN: "3", DisplayName: "V 3"}

	GetDB().CreateUpdateVehicle(v1)
	GetDB().CreateUpdateVehicle(v2)
	GetDB().CreateUpdateVehicle(v3)

	l1 := GetDB().GetVehicles()

	assert.Len(t, l1, 3)

	assert.Equal(t, v1.VIN, l1[0].VIN)
	assert.Equal(t, v2.VIN, l1[1].VIN)
	assert.Equal(t, v3.VIN, l1[2].VIN)
}

func TestDB_CRUDVehicleState(t *testing.T) {
	t.Cleanup(ResetTestDB)
	vehicleID := "123"

	state := GetDB().GetVehicleState(vehicleID)
	assert.Nil(t, state)

	GetDB().SetVehicleStatePluggedIn(vehicleID, true)
	GetDB().SetVehicleStateCharging(vehicleID, ChargeStateChargingOnGrid)
	GetDB().SetVehicleStateSoC(vehicleID, 22)
	GetDB().SetVehicleStateAmps(vehicleID, 5)
	GetDB().SetVehicleStateChargeLimit(vehicleID, 80)
	GetDB().SetVehicleStateIsHome(vehicleID, false)

	state = GetDB().GetVehicleState(vehicleID)
	assert.NotNil(t, state)
	assert.Equal(t, vehicleID, state.VIN)
	assert.Equal(t, true, state.PluggedIn)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	assert.Equal(t, 22, state.SoC)
	assert.Equal(t, 5, state.Amps)
	assert.Equal(t, 80, state.ChargeLimit)
	assert.Equal(t, false, state.IsHome)

	GetDB().SetVehicleStatePluggedIn(vehicleID, false)
	GetDB().SetVehicleStateCharging(vehicleID, ChargeStateNotCharging)
	GetDB().SetVehicleStateSoC(vehicleID, 23)
	GetDB().SetVehicleStateAmps(vehicleID, 6)
	GetDB().SetVehicleStateChargeLimit(vehicleID, 79)
	GetDB().SetVehicleStateIsHome(vehicleID, true)

	state = GetDB().GetVehicleState(vehicleID)
	assert.NotNil(t, state)
	assert.Equal(t, vehicleID, state.VIN)
	assert.Equal(t, false, state.PluggedIn)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	assert.Equal(t, 23, state.SoC)
	assert.Equal(t, 6, state.Amps)
	assert.Equal(t, 79, state.ChargeLimit)
	assert.Equal(t, true, state.IsHome)
}

func TestDB_GetVehicleIDsWithTibberTokenWithoutPricesForTomorrow(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{
		VIN:          "1",
		DisplayName:  "V 1",
		GridProvider: GridProviderTibber,
		TibberToken:  "123",
	}
	GetDB().CreateUpdateVehicle(v)

	now := time.Now().UTC()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for i := 0; i <= 22; i++ {
		SetTibberTestPrice(v.VIN, now.Add(time.Hour*time.Duration(i)), 0.32) // 00:00
	}

	l := GetDB().GetVehicleVINsWithTibberTokenWithoutPricesForTomorrow(45)
	assert.NotNil(t, l)
	assert.Len(t, l, 1)
	assert.Equal(t, v.VIN, l[0])
}

func TestDB_encrypt(t *testing.T) {
	plaintext := "this is a test"
	in := GetDB().encrypt(plaintext)
	out := GetDB().decrypt(in)
	assert.Equal(t, plaintext, out)
}
