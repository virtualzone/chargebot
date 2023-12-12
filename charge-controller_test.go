package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChargeControlGetEstimatedChargeDurationMinutes(t *testing.T) {
	v := &Vehicle{
		TargetSoC: 70,
		MaxAmps:   16,
		NumPhases: 3,
	}
	s := &VehicleState{
		SoC: 50,
	}
	res := ChargeControlGetEstimatedChargeDurationMinutes(v, s)
	assert.Equal(t, res, 109)
}
