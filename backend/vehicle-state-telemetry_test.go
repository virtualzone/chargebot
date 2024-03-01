package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVehicleStateTelemetry_getDistanceFromLatLonInMeters(t *testing.T) {
	lat1 := 50.1083289
	lon1 := 8.6736935

	lat2 := 50.10953
	lon2 := 8.67398

	res := getDistanceFromLatLonInMeters(lat1, lon1, lat2, lon2)
	assert.Equal(t, 135, res)
}

func TestVehicleStateTelemetry_getDistanceFromLatLonInMeters_reverse(t *testing.T) {
	lat1 := 50.1083289
	lon1 := 8.6736935

	lat2 := 50.10953
	lon2 := 8.67398

	res := getDistanceFromLatLonInMeters(lat2, lon2, lat1, lon1)
	assert.Equal(t, 135, res)
}
