package main

import (
	"log"
	"math"
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

	user := GetDB().GetUser(vehicle.UserID)

	if oldState.Amps != telemetryState.Amps {
		GetDB().SetVehicleStateAmps(vehicle.VIN, telemetryState.Amps)
	}
	if oldState.SoC != telemetryState.SoC {
		GetDB().SetVehicleStateSoC(vehicle.VIN, telemetryState.Amps)
	}
	if oldState.ChargeLimit != telemetryState.ChargeLimit {
		GetDB().SetVehicleStateChargeLimit(vehicle.VIN, telemetryState.ChargeLimit)
	}
	if oldState.Charging != ChargeStateNotCharging && !telemetryState.Charging {
		GetDB().SetVehicleStateCharging(vehicle.VIN, ChargeStateNotCharging)
	}
	if oldState.PluggedIn && !telemetryState.PluggedIn {
		t.onVehicleUnplugged(vehicle, oldState)
	}
	if t.isVehicleHome(telemetryState, user) && telemetryState.PluggedIn && !oldState.PluggedIn {
		t.onVehiclePluggedIn(vehicle)
	}
}

func (t *VehicleStateTelemetry) onVehicleUnplugged(vehicle *Vehicle, oldState *VehicleState) {
	// vehicle got plugged out
	GetDB().SetVehicleStatePluggedIn(vehicle.VIN, false)
	GetDB().LogChargingEvent(vehicle.VIN, LogEventVehicleUnplug, "")
	if oldState != nil && oldState.Charging != ChargeStateNotCharging {
		// Vehicle got unplugged while charging
		GetDB().SetVehicleStateCharging(vehicle.VIN, ChargeStateNotCharging)
		GetDB().SetVehicleStateAmps(vehicle.VIN, 0)
	}
}

func (t *VehicleStateTelemetry) onVehiclePluggedIn(vehicle *Vehicle) {
	// vehicle got plugged in at home
	GetDB().SetVehicleStatePluggedIn(vehicle.VIN, true)
	GetDB().LogChargingEvent(vehicle.VIN, LogEventVehiclePlugIn, "")
	if vehicle.Enabled {
		go func() {
			// wait a few moments to ensure vehicle is online
			time.Sleep(10 * time.Second)
			car, err := GetTeslaAPI().InitSession(vehicle, false)
			if err != nil {
				log.Printf("could not init session for vehicle %s on plug in: %s\n", vehicle.VIN, err.Error())
				return
			}
			time.Sleep(5 * time.Second)
			if err := GetTeslaAPI().ChargeStop(car); err != nil {
				log.Printf("could not stop charging for vehicle %s on plug in: %s\n", vehicle.VIN, err.Error())
			}
		}()
	}
}

func (t *VehicleStateTelemetry) isVehicleHome(telemetryState *TelemetryState, user *User) bool {
	dist := t.getDistanceFromLatLonInMeters(user.HomeLatitude, user.HomeLongitude, telemetryState.Latitude, telemetryState.Longitude)
	return dist <= user.HomeRadius
}

func (t *VehicleStateTelemetry) getDistanceFromLatLonInMeters(lat1 float64, lon1 float64, lat2 float64, lon2 float64) int {
	r := 6371 * 1000.0             // Radius of the earth in meters
	dLat := t.deg2rad(lat2 - lat1) // deg2rad below
	dLon := t.deg2rad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(t.deg2rad(lat1))*math.Cos(t.deg2rad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := r * c // Distance in meters
	return int(d)
}

func (t *VehicleStateTelemetry) deg2rad(deg float64) float64 {
	return deg * (math.Pi / 180)
}
