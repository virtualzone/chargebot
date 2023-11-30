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

func (router *UserRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/{token}/surplus", router.recordSurplus).Methods("POST")
	s.HandleFunc("/{token}/surplus", router.getLatestSurpluses).Methods("GET")
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

	res := GetLatestSurplusRecords(vehicleID, 20)
	SendJSON(w, res)
}
