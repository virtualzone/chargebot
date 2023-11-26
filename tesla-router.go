package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type TeslaRouter struct {
}

func (router *TeslaRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/vehicles", router.listVehicles).Methods("GET")
	s.HandleFunc("/my_vehicles", router.myVehicles).Methods("GET")
	s.HandleFunc("/vehicle_add/{id}", router.addVehicle).Methods("POST")
}

func (router *TeslaRouter) listVehicles(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	list, err := TeslaAPIListVehicles(authToken)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendJSON(w, list.Response)
}

func (router *TeslaRouter) myVehicles(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	list := GetVehicles(userID)
	SendJSON(w, list)
}

func (router *TeslaRouter) addVehicle(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vars := mux.Vars(r)
	vehicleId, _ := strconv.Atoi(vars["id"])

	// Check if vehicle belongs to request user
	list, err := TeslaAPIListVehicles(authToken)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	var vehicle *TeslaAPIVehicleEntity = nil
	for _, v := range list.Response {
		if v.VehicleID == vehicleId {
			vehicle = &v
		}
	}

	if vehicle == nil {
		SendBadRequest(w)
		return
	}

	e := &Vehicle{
		ID:          vehicle.VehicleID,
		UserID:      GetUserIDFromRequest(r),
		VIN:         vehicle.VIN,
		DisplayName: vehicle.DisplayName,
	}
	CreateUpdateVehicle(e)
	SendJSON(w, true)
}
