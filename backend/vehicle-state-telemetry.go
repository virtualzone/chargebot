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
		// Only change if charging was not recently started
		event := GetDB().GetLatestChargingEvent(vehicle.VIN, LogEventChargeStart)
		now := time.Now().UTC()
		if event.Timestamp.After(now.Add(-5 * time.Minute)) {
			GetDB().SetVehicleStateCharging(vehicle.VIN, ChargeStateNotCharging)
		}
	}
	user := GetDB().GetUser(vehicle.UserID)
	isVehicleHome := IsVehicleHome(telemetryState, user)
	if oldState.IsHome != isVehicleHome {
		GetDB().SetVehicleStateIsHome(vehicle.VIN, isVehicleHome)
	}

	if vehicle.Enabled && oldState.Charging == ChargeStateNotCharging && telemetryState.Charging {
		// if vehicle is charging although assumed not to, it could be that it has been plugged in recently
		if !oldState.PluggedIn && isVehicleHome {
			OnVehiclePluggedIn(vehicle)
			return
		} else {
			// otherwise, this is an anomaly where chargebot stopped charging but vehicle is still charging
			// check if charging was actually stopped within the last minutes (else, it might just be the A/C)
			event := GetDB().GetLatestChargingEvent(vehicle.VIN, LogEventChargeStop)
			now := time.Now().UTC()
			if event.Timestamp.After(now.Add(-5 * time.Minute)) {
				log.Printf("Anomaly detected: Vehicle %s was assumed to be not charging, but actually is - stopping it\n", vehicle.VIN)
				GetChargeController().stopCharging(vehicle)
			}
		}
	}
	// Workarounds for incorrect ChargeState in telemetry data
	// https://github.com/teslamotors/fleet-telemetry/issues/123
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
