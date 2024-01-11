package main

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ManualControlRouter struct {
}

type ManualControlResponse struct {
	Result bool   `json:"res"`
	Error  string `json:"err"`
}

func (router *ManualControlRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/{id}/wakeUp", router.wakeUp).Methods("POST")
	s.HandleFunc("/{id}/chargeStart", router.chargeStart).Methods("POST")
	s.HandleFunc("/{id}/chargeStop", router.chargeStop).Methods("POST")
	s.HandleFunc("/{id}/chargeLimit/{limit}", router.chargeLimit).Methods("POST")
	s.HandleFunc("/{id}/chargeAmps/{amps}", router.chargeAmps).Methods("POST")
	s.HandleFunc("/{id}/scheduledCharging/{enabled}/{mins}", router.scheduledCharging).Methods("POST")
}

func (router *ManualControlRouter) wakeUp(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	err := GetTeslaAPI().WakeUpVehicle(authToken, vehicle)
	router.sendResponse(w, true, err)
}

func (router *ManualControlRouter) chargeStart(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	b, err := GetTeslaAPI().ChargeStart(authToken, vehicle)
	router.sendResponse(w, b, err)
}

func (router *ManualControlRouter) chargeStop(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	b, err := GetTeslaAPI().ChargeStop(authToken, vehicle)
	router.sendResponse(w, b, err)
}

func (router *ManualControlRouter) chargeLimit(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	limit, _ := strconv.Atoi(mux.Vars(r)["limit"])
	b, err := GetTeslaAPI().SetChargeLimit(authToken, vehicle, limit)
	router.sendResponse(w, b, err)
}

func (router *ManualControlRouter) chargeAmps(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	amps, _ := strconv.Atoi(mux.Vars(r)["amps"])
	b, err := GetTeslaAPI().SetChargeAmps(authToken, vehicle, amps)
	router.sendResponse(w, b, err)
}

func (router *ManualControlRouter) scheduledCharging(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	enabled, _ := mux.Vars(r)["enabled"]
	mins, _ := strconv.Atoi(mux.Vars(r)["mins"])
	b, err := GetTeslaAPI().SetScheduledCharging(authToken, vehicle, enabled == "1", mins)
	router.sendResponse(w, b, err)
}

func (router *ManualControlRouter) getVehicleFromRequest(r *http.Request) *Vehicle {
	vars := mux.Vars(r)
	vehicleId, _ := strconv.Atoi(vars["id"])

	vehicle := GetDB().GetVehicleByID(vehicleId)
	if vehicle == nil || vehicle.UserID != GetUserIDFromRequest(r) {
		return nil
	}
	return vehicle
}

func (router *ManualControlRouter) sendResponse(w http.ResponseWriter, b bool, err error) {
	errTxt := ""
	if err != nil {
		errTxt = err.Error()
	}
	res := ManualControlResponse{
		Result: b,
		Error:  errTxt,
	}
	SendJSON(w, res)
}
