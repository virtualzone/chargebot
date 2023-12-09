package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type UserRouter struct{}

type SurplusRecordingRequest struct {
	Password            string `json:"password"`
	SurplusWatts        int    `json:"surplus_watts"`
	InverterActivePower int    `json:"inverter_active_power_watts"`
	Consumption         int    `json:"consumption_watts"`
}

type SurplusRecordingResponse struct {
}

type PlugInOutRequest struct {
	Password string `json:"password"`
}

func (router *UserRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/{token}/surplus", router.recordSurplus).Methods("POST")
	s.HandleFunc("/{token}/surplus", router.getLatestSurpluses).Methods("GET")
	s.HandleFunc("/{token}/plugged_in", router.vehiclePluggedIn).Methods("POST")
	s.HandleFunc("/{token}/unplugged", router.vehicleUnplugged).Methods("POST")
	s.HandleFunc("/{token}/events", router.getLatestChargingEvents).Methods("GET")
}

func (router *UserRouter) recordSurplus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	var m *SurplusRecordingRequest
	if err := UnmarshalBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}

	if !IsTokenPasswordValid(token, m.Password) {
		SendUnauthorized(w)
		return
	}

	vehicleID := GetAPITokenVehicleID(token)
	if vehicleID == 0 {
		SendBadRequest(w)
		return
	}

	vehicle := GetVehicleByID(vehicleID)
	if vehicle == nil {
		SendInternalServerError(w)
		return
	}

	surplus := m.SurplusWatts
	if surplus == 0 {
		if m.InverterActivePower != 0 || m.Consumption != 0 {
			surplus = m.InverterActivePower - m.Consumption
		}
	}

	RecordSurplus(vehicle.ID, surplus)
	SendJSON(w, true)
}

func (router *UserRouter) getLatestSurpluses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	vehicleID := GetAPITokenVehicleID(token)
	if vehicleID == 0 {
		SendBadRequest(w)
		return
	}

	res := GetLatestSurplusRecords(vehicleID, 50)
	SendJSON(w, res)
}

func (router *UserRouter) getLatestChargingEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	vehicleID := GetAPITokenVehicleID(token)
	if vehicleID == 0 {
		SendBadRequest(w)
		return
	}

	res := GetLatestChargingEvents(vehicleID, 50)
	SendJSON(w, res)
}

func (router *UserRouter) updateVehiclePlugState(w http.ResponseWriter, r *http.Request, pluggedIn bool) {
	vars := mux.Vars(r)
	token := vars["token"]

	var m *SurplusRecordingRequest
	if err := UnmarshalBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}

	if !IsTokenPasswordValid(token, m.Password) {
		SendUnauthorized(w)
		return
	}

	vehicleID := GetAPITokenVehicleID(token)
	if vehicleID == 0 {
		SendBadRequest(w)
		return
	}

	vehicle := GetVehicleByID(vehicleID)

	authToken := TeslaAPIGetCachedAccessToken(vehicle.UserID)
	if authToken != "" {
		data, err := TeslaAPIGetVehicleData(authToken, vehicleID)
		if err != nil {
			log.Println(err)
			LogChargingEvent(vehicleID, LogEventVehicleUpdateData, err.Error())
		} else {
			SetVehicleStateSoC(vehicleID, data.ChargeState.BatteryLevel)
			LogChargingEvent(vehicleID, LogEventVehicleUpdateData, fmt.Sprintf("vehicle SoC updated: %d", data.ChargeState.BatteryLevel))
		}
	} else {
		log.Printf("could not get access token to update vehicle data on plug in/out for vehicle id %d\n", vehicleID)
	}

	SetVehicleStatePluggedIn(vehicleID, pluggedIn)
	if pluggedIn {
		LogChargingEvent(vehicleID, LogEventVehiclePlugIn, "")
	} else {
		LogChargingEvent(vehicleID, LogEventVehicleUnplug, "")
	}
}

func (router *UserRouter) vehiclePluggedIn(w http.ResponseWriter, r *http.Request) {
	router.updateVehiclePlugState(w, r, true)
}

func (router *UserRouter) vehicleUnplugged(w http.ResponseWriter, r *http.Request) {
	router.updateVehiclePlugState(w, r, false)
}
