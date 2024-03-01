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

	//user := GetDB().GetUser(vehicle.UserID)

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
	/*
		if oldState.PluggedIn && !telemetryState.PluggedIn {
			t.onVehicleUnplugged(vehicle, oldState)
		}
		if t.isVehicleHome(telemetryState, user) && telemetryState.PluggedIn && !oldState.PluggedIn {
			t.onVehiclePluggedIn(vehicle)
		}
	*/
}
