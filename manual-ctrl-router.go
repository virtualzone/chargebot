package main

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ManualControlRouter struct {
}

type ManualControlResponse struct {
	Error string `json:"err"`
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

	_, err := GetTeslaAPI().InitSession(authToken, vehicle, true)
	router.sendResponse(w, err)
}

func (router *ManualControlRouter) chargeStart(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	car, err := GetTeslaAPI().InitSession(authToken, vehicle, false)
	if err != nil {
		router.sendResponse(w, err)
		return
	}
	err = GetTeslaAPI().ChargeStart(car)
	router.sendResponse(w, err)
}

func (router *ManualControlRouter) chargeStop(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	car, err := GetTeslaAPI().InitSession(authToken, vehicle, false)
	if err != nil {
		router.sendResponse(w, err)
		return
	}
	err = GetTeslaAPI().ChargeStop(car)
	router.sendResponse(w, err)
}

func (router *ManualControlRouter) chargeLimit(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	limit, _ := strconv.Atoi(mux.Vars(r)["limit"])
	car, err := GetTeslaAPI().InitSession(authToken, vehicle, false)
	if err != nil {
		router.sendResponse(w, err)
		return
	}
	err = GetTeslaAPI().SetChargeLimit(car, limit)
	router.sendResponse(w, err)
}

func (router *ManualControlRouter) chargeAmps(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	amps, _ := strconv.Atoi(mux.Vars(r)["amps"])
	car, err := GetTeslaAPI().InitSession(authToken, vehicle, false)
	if err != nil {
		router.sendResponse(w, err)
		return
	}
	err = GetTeslaAPI().SetChargeAmps(car, amps)
	router.sendResponse(w, err)
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
	car, err := GetTeslaAPI().InitSession(authToken, vehicle, false)
	if err != nil {
		router.sendResponse(w, err)
		return
	}
	err = GetTeslaAPI().SetScheduledCharging(car, enabled == "1", mins)
	router.sendResponse(w, err)
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

func (router *ManualControlRouter) sendResponse(w http.ResponseWriter, err error) {
	errTxt := ""
	if err != nil {
		errTxt = err.Error()
	}
	res := ManualControlResponse{
		Error: errTxt,
	}
	SendJSON(w, res)
}
