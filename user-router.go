package main

import (
	"log"
	"net/http"
	"time"

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
	s.HandleFunc("/{token}/state", router.getVehicleState).Methods("GET")
	s.HandleFunc("/{token}/surplus", router.recordSurplus).Methods("POST")
	s.HandleFunc("/{token}/surplus", router.getLatestSurpluses).Methods("GET")
	s.HandleFunc("/{token}/plugged_in", router.vehiclePluggedIn).Methods("POST")
	s.HandleFunc("/{token}/unplugged", router.vehicleUnplugged).Methods("POST")
	s.HandleFunc("/{token}/events", router.getLatestChargingEvents).Methods("GET")
}

func (router *UserRouter) getVehicleState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	vehicleID := GetDB().GetAPITokenVehicleID(token)
	if vehicleID == 0 {
		SendBadRequest(w)
		return
	}

	state := GetDB().GetVehicleState(vehicleID)
	if state == nil {
		SendNotFound(w)
		return
	}
	SendJSON(w, state)
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

	vehicleID := GetDB().GetAPITokenVehicleID(token)
	if vehicleID == 0 {
		SendBadRequest(w)
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

func (router *UserRouter) getLatestSurpluses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	vehicleID := GetDB().GetAPITokenVehicleID(token)
	if vehicleID == 0 {
		SendBadRequest(w)
		return
	}

	res := GetDB().GetLatestSurplusRecords(vehicleID, 50)
	SendJSON(w, res)
}

func (router *UserRouter) getLatestChargingEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	vehicleID := GetDB().GetAPITokenVehicleID(token)
	if vehicleID == 0 {
		SendBadRequest(w)
		return
	}

	res := GetDB().GetLatestChargingEvents(vehicleID, 50)
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

	if !GetDB().IsTokenPasswordValid(token, m.Password) {
		SendUnauthorized(w)
		return
	}

	vehicleID := GetDB().GetAPITokenVehicleID(token)
	if vehicleID == 0 {
		SendBadRequest(w)
		return
	}

	vehicle := GetDB().GetVehicleByID(vehicleID)

	GetDB().SetVehicleStatePluggedIn(vehicleID, pluggedIn)
	if pluggedIn {
		GetDB().LogChargingEvent(vehicleID, LogEventVehiclePlugIn, "")
	} else {
		GetDB().LogChargingEvent(vehicleID, LogEventVehicleUnplug, "")
	}

	authToken := GetTeslaAPI().GetOrRefreshAccessToken(vehicle.UserID)
	if authToken == "" {
		log.Printf("could not get access token to update vehicle data on plug in/out for vehicle id %d\n", vehicleID)
	} else {
		if pluggedIn && vehicle.Enabled {
			go func() {
				// wait a few moments to ensure vehicle is online
				time.Sleep(10 * time.Second)
				UpdateVehicleDataSaveSoC(authToken, vehicle)
				if _, err := GetTeslaAPI().SetChargeLimit(authToken, vehicle, 50); err != nil {
					log.Printf("could not set charge limit for vehicle %d on plug in: %s\n", vehicleID, err.Error())
				}
				time.Sleep(5 * time.Second)
				if _, err := GetTeslaAPI().ChargeStop(authToken, vehicle); err != nil {
					log.Printf("could not stop charging for vehicle %d on plug in: %s\n", vehicleID, err.Error())
				}
			}()
		}
	}
}

func (router *UserRouter) vehiclePluggedIn(w http.ResponseWriter, r *http.Request) {
	router.updateVehiclePlugState(w, r, true)
}

func (router *UserRouter) vehicleUnplugged(w http.ResponseWriter, r *http.Request) {
	router.updateVehiclePlugState(w, r, false)
}
