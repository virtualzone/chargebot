package main

import (
	"log"
	"time"

	. "github.com/virtualzone/chargebot/goshared"
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

	user := GetDB().GetUser(vehicle.UserID)
	isVehicleHome := IsVehicleHome(telemetryState, user)

	s := &PersistedTelemetryState{
		VIN:         telemetryState.VIN,
		PluggedIn:   telemetryState.PluggedIn,
		Charging:    telemetryState.Charging,
		SoC:         telemetryState.SoC,
		Amps:        telemetryState.Amps,
		ChargeLimit: telemetryState.ChargeLimit,
		IsHome:      isVehicleHome,
		UTC:         time.Now().UTC().Unix(),
	}

	GetTelemetryQueue().SetState(s)
	GetDB().SaveTelemetryState(s)
}
