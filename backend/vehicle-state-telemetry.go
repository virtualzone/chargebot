package main

import (
	"log"
	"strings"
	"time"
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
	// Handle anomaly where chargebot stopped charging but vehicle is still charging
	if vehicle.Enabled && oldState.Charging == ChargeStateNotCharging && telemetryState.Charging {
		log.Printf("Anomaly detected: Vehicle %s was assumed to be not charging, but actually is - stopping it\n", vehicle.VIN)
		GetChargeController().stopCharging(vehicle)
	}
	// Workarounds for incorrect ChargeState in telemetry data
	// https://github.com/teslamotors/fleet-telemetry/issues/123
	user := GetDB().GetUser(vehicle.UserID)
	isVehicleHome := IsVehicleHome(telemetryState, user)
	if oldState.PluggedIn && !isVehicleHome {
		// If vehicle was plugged in but is not home anymore, it is obiously not plugged in anymore
		OnVehicleUnplugged(vehicle, oldState)
		return
	}
	if !isVehicleHome {
		return
	}
	now := time.Now().UTC()
	if CanUpdateVehicleData(vehicle.VIN, &now) {
		data, err := GetTeslaAPI().GetVehicleData(vehicle)
		if err != nil {
			log.Println(err)
			return
		}
		GetDB().LogChargingEvent(vehicle.VIN, LogEventVehicleUpdateData, "")
		cableConnected := (strings.ToLower(data.ChargeState.ConnectedChargeCable) == "iec" || strings.ToLower(data.ChargeState.ConnectedChargeCable) == "sae")
		if oldState.PluggedIn && !cableConnected {
			OnVehicleUnplugged(vehicle, oldState)
		}
		if !oldState.PluggedIn && cableConnected {
			OnVehiclePluggedIn(vehicle)
		}
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
