package main

import (
	"log"
)

type VehicleStateTelemetry struct{}

type TelemetryState struct {
	VIN         string
	PluggedIn   bool
	Charging    bool
	ChargeLimit int
	SoC         int
	Amps        int
	Latitude    float64
	Longitude   float64
}

func (t *VehicleStateTelemetry) Update(args *TelemetryState, reply *bool) error {
	t.updateVehicleState(args)
	*reply = true
	return nil
}

func (t *VehicleStateTelemetry) updateVehicleState(telemetryState *TelemetryState) {
	vehicle := GetDB().GetVehicleByVIN(telemetryState.VIN)
	if vehicle == nil {
		log.Printf("could not find vehicle by vin for telemetry data: %s\n", telemetryState.VIN)
		return
	}
	oldState := GetDB().GetVehicleState(vehicle.VIN)
	if oldState == nil {
		oldState = &VehicleState{
			PluggedIn: false,
			Charging:  ChargeStateNotCharging,
			Amps:      -1,
			SoC:       -1,
		}
	}

	if oldState.Amps != telemetryState.Amps {
		GetDB().SetVehicleStateAmps(vehicle.VIN, telemetryState.Amps)
	}
	if oldState.SoC != telemetryState.SoC {
		GetDB().SetVehicleStateSoC(vehicle.VIN, telemetryState.SoC)
	}
	if oldState.ChargeLimit != telemetryState.ChargeLimit {
		GetDB().SetVehicleStateChargeLimit(vehicle.VIN, telemetryState.ChargeLimit)
	}
	if oldState.Charging != ChargeStateNotCharging && !telemetryState.Charging {
		GetDB().SetVehicleStateCharging(vehicle.VIN, ChargeStateNotCharging)
	}
	// Workarounds for incorrect ChargeState in telemetry data
	user := GetDB().GetUser(vehicle.UserID)
	if oldState.PluggedIn && !IsVehicleHome(telemetryState, user) {
		// If vehicle was plugged in but is not home anymore, it is obiously not plugged in anymore
		OnVehicleUnplugged(vehicle, oldState)
		return
	}
	// Try to get data from vehicle, but do NOT wake it
	/*
		if CanUpdateVehicleData(vehicle.VIN, time.Now().UTC()) {

		}
		data, err := GetTeslaAPI().GetVehicleData(vehicle)
		if err != nil {
		}
	*/

	/*
		if oldState.PluggedIn && !telemetryState.PluggedIn {
			t.onVehicleUnplugged(vehicle, oldState)
		}
		if t.isVehicleHome(telemetryState, user) && telemetryState.PluggedIn && !oldState.PluggedIn {
			t.onVehiclePluggedIn(vehicle)
		}
	*/
}
