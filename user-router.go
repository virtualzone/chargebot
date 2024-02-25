package main

import (
	"log"
	"net/http"
	"strconv"
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
	s.HandleFunc("/{token}/{id}/state", router.getVehicleState).Methods("GET")
	s.HandleFunc("/{token}/{id}/surplus", router.recordSurplus).Methods("POST")
	s.HandleFunc("/{token}/{id}/surplus", router.getLatestSurpluses).Methods("GET")
	s.HandleFunc("/{token}/{id}/plugged_in", router.vehiclePluggedIn).Methods("POST")
	s.HandleFunc("/{token}/{id}/unplugged", router.vehicleUnplugged).Methods("POST")
	s.HandleFunc("/{token}/{id}/events", router.getLatestChargingEvents).Methods("GET")
}

func (router *UserRouter) getVehicleState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	vehicleIDString := vars["id"]
	vehicleID, err := strconv.Atoi(vehicleIDString)
	if err != nil {
		SendBadRequest(w)
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

func (router *UserRouter) getLatestSurpluses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	vehicleIDString := vars["id"]
	vehicleID, err := strconv.Atoi(vehicleIDString)
	if err != nil {
		SendBadRequest(w)
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

	res := GetDB().GetLatestSurplusRecords(vehicleID, 50)
	SendJSON(w, res)
}

func (router *UserRouter) getLatestChargingEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	vehicleIDString := vars["id"]
	vehicleID, err := strconv.Atoi(vehicleIDString)
	if err != nil {
		SendBadRequest(w)
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

	res := GetDB().GetLatestChargingEvents(vehicleID, 50)
	SendJSON(w, res)
}

func (router *UserRouter) updateVehiclePlugState(w http.ResponseWriter, r *http.Request, pluggedIn bool) {
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

	GetDB().SetVehicleStatePluggedIn(vehicleID, pluggedIn)
	if pluggedIn {
		GetDB().LogChargingEvent(vehicleID, LogEventVehiclePlugIn, "")
	} else {
		GetDB().LogChargingEvent(vehicleID, LogEventVehicleUnplug, "")
	}

	if pluggedIn && vehicle.Enabled {
		go func() {
			// wait a few moments to ensure vehicle is online
			time.Sleep(10 * time.Second)
			car, err := GetTeslaAPI().InitSession(vehicle, false)
			if err != nil {
				log.Printf("could not init session for vehicle %d on plug in: %s\n", vehicleID, err.Error())
				return
			}
			UpdateVehicleDataSaveSoC(vehicle)
			if err := GetTeslaAPI().SetChargeLimit(car, 50); err != nil {
				log.Printf("could not set charge limit for vehicle %d on plug in: %s\n", vehicleID, err.Error())
			}
			time.Sleep(5 * time.Second)
			if err := GetTeslaAPI().ChargeStop(car); err != nil {
				log.Printf("could not stop charging for vehicle %d on plug in: %s\n", vehicleID, err.Error())
			}
		}()
	}
	if !pluggedIn && vehicle.Enabled {
		state := GetDB().GetVehicleState(vehicle.ID)
		if state != nil && state.Charging != ChargeStateNotCharging {
			// Vehicle got unplugged while charging
			GetDB().SetVehicleStateCharging(vehicle.ID, ChargeStateNotCharging)
			GetDB().SetVehicleStateAmps(vehicle.ID, 0)
		}
	}
}

func (router *UserRouter) vehiclePluggedIn(w http.ResponseWriter, r *http.Request) {
	router.updateVehiclePlugState(w, r, true)
}

func (router *UserRouter) vehicleUnplugged(w http.ResponseWriter, r *http.Request) {
	router.updateVehiclePlugState(w, r, false)
}
