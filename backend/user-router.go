package main

import (
	"net/http"
	"strconv"

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
	s.HandleFunc("/{token}/{id}/surplus", router.recordSurplus).Methods("POST")
}

func (router *UserRouter) recordSurplus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	vehicleIDString := vars["id"]
	vehicleID, err := strconv.Atoi(vehicleIDString)
	if err != nil {
		SendBadRequest(w)
		return
	}

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

	if !GetDB().IsUserOwnerOfVehicle(userID, vehicleID) {
		SendForbidden(w)
		return
	}

	vehicle := GetDB().GetVehicleByID(vehicleID)
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

	GetDB().RecordSurplus(vehicle.ID, surplus)
	SendJSON(w, true)
}
