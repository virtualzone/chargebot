package main

import (
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
	s.HandleFunc("/{token}/{vin}/plugged_in", router.vehiclePluggedIn).Methods("POST")
	s.HandleFunc("/{token}/{vin}/unplugged", router.vehicleUnplugged).Methods("POST")
}

func (router *UserRouter) recordSurplus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	var m *SurplusRecordingRequest
	if err := UnmarshalBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}

	if !GetDB().IsTokenPasswordValid(token, m.Password) {
		SendUnauthorized(w)
		return
	}

	userID := GetDB().GetAPITokenUserID(token)
	if userID == "" {
		SendBadRequest(w)
		return
	}

	surplus := m.SurplusWatts
	if surplus == 0 {
		if m.InverterActivePower != 0 || m.Consumption != 0 {
			surplus = m.InverterActivePower - m.Consumption
		}
	}

	GetDB().RecordSurplus(userID, surplus)
	SendJSON(w, true)
}

func (router *UserRouter) updateVehiclePlugState(w http.ResponseWriter, r *http.Request, pluggedIn bool) {
	vars := mux.Vars(r)
	token := vars["token"]
	vin := vars["vin"]

	var m *SurplusRecordingRequest
	if err := UnmarshalBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}

	if !GetDB().IsTokenPasswordValid(token, m.Password) {
		SendUnauthorized(w)
		return
	}

	userID := GetDB().GetAPITokenUserID(token)
	if userID == "" {
		SendBadRequest(w)
		return
	}

	if !GetDB().IsUserOwnerOfVehicle(userID, vin) {
		SendForbidden(w)
		return
	}

	vehicle := GetDB().GetVehicleByVIN(vin)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	if pluggedIn {
		OnVehiclePluggedIn(vehicle)
	} else {
		state := GetDB().GetVehicleState(vehicle.VIN)
		OnVehicleUnplugged(vehicle, state)
	}
}

func (router *UserRouter) vehiclePluggedIn(w http.ResponseWriter, r *http.Request) {
	router.updateVehiclePlugState(w, r, true)
}

func (router *UserRouter) vehicleUnplugged(w http.ResponseWriter, r *http.Request) {
	router.updateVehiclePlugState(w, r, false)
}
