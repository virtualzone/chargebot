package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type TeslaRouter struct {
}

type CreateAPITokenRequest struct {
	VehicleID int `json:"vehicle_id"`
}

type GetAPITokenResponse struct {
	Token     string `json:"token"`
	VehicleID int    `json:"vehicle_id"`
	Password  string `json:"password"`
}

func (router *TeslaRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/vehicles", router.listVehicles).Methods("GET")
	s.HandleFunc("/my_vehicles", router.myVehicles).Methods("GET")
	s.HandleFunc("/vehicle_add/{id}", router.addVehicle).Methods("POST")
	s.HandleFunc("/vehicle_update/{id}", router.updateVehicle).Methods("PUT")
	s.HandleFunc("/vehicle_delete/{id}", router.deleteVehicle).Methods("DELETE")
	s.HandleFunc("/api_token_create", router.createAPIToken).Methods("POST")
	s.HandleFunc("/api_token_update/{id}", router.updateAPIToken).Methods("POST")
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
		ID:              vehicle.VehicleID,
		UserID:          GetUserIDFromRequest(r),
		VIN:             vehicle.VIN,
		DisplayName:     vehicle.DisplayName,
		Enabled:         false,
		TargetSoC:       70,
		MaxAmps:         16,
		SurplusCharging: true,
		MinChargeTime:   15,
		MinSurplus:      2000,
		LowcostCharging: false,
		MaxPrice:        20,
		TibberToken:     "",
	}
	CreateUpdateVehicle(e)
	SendJSON(w, true)
}

func (router *TeslaRouter) updateVehicle(w http.ResponseWriter, r *http.Request) {
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

	var m *Vehicle
	UnmarshalValidateBody(r.Body, &m)

	e := &Vehicle{
		ID:              vehicle.VehicleID,
		UserID:          GetUserIDFromRequest(r),
		VIN:             vehicle.VIN,
		DisplayName:     vehicle.DisplayName,
		Enabled:         m.Enabled,
		TargetSoC:       m.TargetSoC,
		MaxAmps:         m.MaxAmps,
		SurplusCharging: m.SurplusCharging,
		MinChargeTime:   m.MinChargeTime,
		MinSurplus:      m.MinSurplus,
		LowcostCharging: m.LowcostCharging,
		MaxPrice:        m.MaxPrice,
		TibberToken:     m.TibberToken,
	}
	CreateUpdateVehicle(e)
	SendJSON(w, true)
}

func (router *TeslaRouter) deleteVehicle(w http.ResponseWriter, r *http.Request) {
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

	DeleteVehicle(vehicle.VehicleID)
	SendJSON(w, true)
}

func (router *TeslaRouter) createAPIToken(w http.ResponseWriter, r *http.Request) {
	var m CreateAPITokenRequest
	err := UnmarshalValidateBody(r.Body, &m)
	if err != nil {
		SendBadRequest(w)
		return
	}

	userID := GetUserIDFromRequest(r)
	if !IsUserOwnerOfVehicle(userID, m.VehicleID) {
		SendForbidden(w)
		return
	}

	password := GeneratePassword(16, true, true)
	token := CreateAPIToken(m.VehicleID, password)

	resp := GetAPITokenResponse{
		Token:     token,
		VehicleID: m.VehicleID,
		Password:  password,
	}
	SendJSON(w, resp)
}

func (router *TeslaRouter) updateAPIToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["id"]
	userID := GetUserIDFromRequest(r)
	vehicleID := GetAPITokenVehicleID(token)
	if !IsUserOwnerOfVehicle(userID, vehicleID) {
		SendForbidden(w)
		return
	}

	password := GeneratePassword(16, true, true)
	UpdateAPITokenPassword(token, password)

	resp := GetAPITokenResponse{
		Token:     token,
		VehicleID: vehicleID,
		Password:  password,
	}
	SendJSON(w, resp)
}
