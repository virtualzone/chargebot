package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

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
	list, err := GetTeslaAPI().ListVehicles(authToken)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendJSON(w, list)
}

func (router *TeslaRouter) myVehicles(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	list := GetDB().GetVehicles(userID)
	SendJSON(w, list)
}

func (router *TeslaRouter) addVehicle(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vars := mux.Vars(r)
	vehicleId, _ := strconv.Atoi(vars["id"])

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles(authToken)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	var vehicle *TeslaAPIVehicleEntity = nil
	for _, v := range list {
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
	GetDB().CreateUpdateVehicle(e)
	SendJSON(w, true)
}

func (router *TeslaRouter) updateVehicle(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vars := mux.Vars(r)
	vehicleId, _ := strconv.Atoi(vars["id"])

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles(authToken)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	var vehicle *TeslaAPIVehicleEntity = nil
	for _, v := range list {
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

	eOld := GetDB().GetVehicleByID(vehicle.VehicleID)

	e := &Vehicle{
		ID:              vehicle.VehicleID,
		UserID:          GetUserIDFromRequest(r),
		VIN:             vehicle.VIN,
		DisplayName:     vehicle.DisplayName,
		Enabled:         m.Enabled,
		TargetSoC:       m.TargetSoC,
		MaxAmps:         m.MaxAmps,
		NumPhases:       m.NumPhases,
		SurplusCharging: m.SurplusCharging,
		MinChargeTime:   m.MinChargeTime,
		MinSurplus:      m.MinSurplus,
		LowcostCharging: m.LowcostCharging,
		MaxPrice:        m.MaxPrice,
		TibberToken:     m.TibberToken,
	}
	GetDB().CreateUpdateVehicle(e)

	// If vehicle was not enabled, but is enabled now, update current SoC
	if (eOld != nil) && (e.Enabled) && (!eOld.Enabled) {
		go func() {
			car, err := GetTeslaAPI().InitSession(authToken, e)
			if err != nil {
				log.Printf("could not init session for vehicle %d on plug in: %s\n", e.ID, err.Error())
				return
			}
			UpdateVehicleDataSaveSoC(authToken, e)
			if err := GetTeslaAPI().SetChargeLimit(car, 50); err != nil {
				log.Printf("could not set charge limit for vehicle %d on plug in: %s\n", e.ID, err.Error())
			}
			time.Sleep(5 * time.Second)
			if err := GetTeslaAPI().ChargeStop(car); err != nil {
				log.Printf("could not stop charging for vehicle %d on plug in: %s\n", e.ID, err.Error())
			}
		}()
	}

	SendJSON(w, true)
}

func (router *TeslaRouter) deleteVehicle(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vars := mux.Vars(r)
	vehicleId, _ := strconv.Atoi(vars["id"])

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles(authToken)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	var vehicle *TeslaAPIVehicleEntity = nil
	for _, v := range list {
		if v.VehicleID == vehicleId {
			vehicle = &v
		}
	}

	if vehicle == nil {
		SendBadRequest(w)
		return
	}

	GetDB().DeleteVehicle(vehicle.VehicleID)
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
	if !GetDB().IsUserOwnerOfVehicle(userID, m.VehicleID) {
		SendForbidden(w)
		return
	}

	password := GeneratePassword(16, true, true)
	token := GetDB().CreateAPIToken(m.VehicleID, password)

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
	vehicleID := GetDB().GetAPITokenVehicleID(token)
	if !GetDB().IsUserOwnerOfVehicle(userID, vehicleID) {
		SendForbidden(w)
		return
	}

	password := GeneratePassword(16, true, true)
	GetDB().UpdateAPITokenPassword(token, password)

	resp := GetAPITokenResponse{
		Token:     token,
		VehicleID: vehicleID,
		Password:  password,
	}
	SendJSON(w, resp)
}
