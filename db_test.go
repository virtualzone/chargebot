package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_SetVehicleStates(t *testing.T) {
	t.Cleanup(ResetTestDB)

	GetDB().SetVehicleStateCharging(123, ChargeStateChargingOnSolar)
	GetDB().SetVehicleStateAmps(123, 5)
	GetDB().SetVehicleStatePluggedIn(123, true)
	GetDB().SetVehicleStateSoC(123, 55)

	state := GetDB().GetVehicleState(123)
	assert.NotNil(t, state)
	assert.Equal(t, ChargeStateChargingOnSolar, state.Charging)
	assert.Equal(t, 5, state.Amps)
	assert.Equal(t, 55, state.SoC)
	assert.Equal(t, true, state.PluggedIn)

}
