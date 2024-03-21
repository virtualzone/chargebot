package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	. "github.com/virtualzone/chargebot/goshared"
)

type TeslaRouter struct {
}

type GetAPITokenResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

func (router *TeslaRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/my_vehicles", router.myVehicles).Methods("GET")
	s.HandleFunc("/vehicle_add/{vin}", router.addVehicle).Methods("POST")
	s.HandleFunc("/vehicle_delete/{vin}", router.deleteVehicle).Methods("DELETE")
	s.HandleFunc("/api_token_create", router.createAPIToken).Methods("POST")
	s.HandleFunc("/api_token_update/{id}", router.updateAPIToken).Methods("POST")
}

func (router *TeslaRouter) myVehicles(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	list := GetDB().GetVehicles(userID)
	SendJSON(w, list)
}

func (router *TeslaRouter) addVehicle(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	vars := mux.Vars(r)
	vin := vars["vin"]

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles(userID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	var vehicle *TeslaAPIVehicleEntity = nil
	for _, v := range list {
		if v.VIN == vin {
			vehicle = &v
		}
	}

	if vehicle == nil {
		SendBadRequest(w)
		return
	}

	e := &Vehicle{
		VIN:    vehicle.VIN,
		UserID: userID,
	}
	GetDB().CreateUpdateVehicle(e)

	SendJSON(w, true)
}

func (router *TeslaRouter) deleteVehicle(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	vars := mux.Vars(r)
	vin := vars["vin"]

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles(userID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	var vehicle *TeslaAPIVehicleEntity = nil
	for _, v := range list {
		if v.VIN == vin {
			vehicle = &v
		}
	}

	if vehicle == nil {
		SendBadRequest(w)
		return
	}

	e := GetDB().GetVehicleByVIN(vin)
	if e == nil {
		SendBadRequest(w)
		return
	}

	GetDB().DeleteVehicle(vehicle.VIN)
	SendJSON(w, true)
}

func (router *TeslaRouter) createAPIToken(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)

	password := GeneratePassword(16, true, true)
	token := GetDB().CreateAPIToken(userID, password)

	resp := GetAPITokenResponse{
		Token:    token,
		UserID:   userID,
		Password: password,
	}
	SendJSON(w, resp)
}

func (router *TeslaRouter) updateAPIToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["id"]
	userID := GetUserIDFromRequest(r)
	userIDToken := GetDB().GetAPITokenUserID(token)
	if userID != userIDToken {
		SendForbidden(w)
		return
	}

	password := GeneratePassword(16, true, true)
	GetDB().UpdateAPITokenPassword(token, password)

	resp := GetAPITokenResponse{
		Token:    token,
		UserID:   userID,
		Password: password,
	}
	SendJSON(w, resp)
}
