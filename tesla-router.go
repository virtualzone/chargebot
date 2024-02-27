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

type GetAPITokenResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

func (router *TeslaRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/vehicles", router.listVehicles).Methods("GET")
	s.HandleFunc("/my_vehicles", router.myVehicles).Methods("GET")
	s.HandleFunc("/vehicle_add/{id}", router.addVehicle).Methods("POST")
	s.HandleFunc("/vehicle_update/{id}", router.updateVehicle).Methods("PUT")
	s.HandleFunc("/vehicle_delete/{id}", router.deleteVehicle).Methods("DELETE")
	s.HandleFunc("/api_token_create", router.createAPIToken).Methods("POST")
	s.HandleFunc("/api_token_update/{id}", router.updateAPIToken).Methods("POST")
	s.HandleFunc("/state/{id}", router.getVehicleState).Methods("GET")
	s.HandleFunc("/surplus/{id}", router.getLatestSurpluses).Methods("GET")
	s.HandleFunc("/events/{id}", router.getLatestChargingEvents).Methods("GET")
}

func (router *TeslaRouter) listVehicles(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	list, err := GetTeslaAPI().ListVehicles(userID)
	if err != nil {
		log.Println(err)
		SendUnauthorized(w)
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
	userID := GetUserIDFromRequest(r)
	vars := mux.Vars(r)
	vehicleId, _ := strconv.Atoi(vars["id"])

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles(userID)
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
		UserID:          userID,
		VIN:             vehicle.VIN,
		DisplayName:     vehicle.DisplayName,
		Enabled:         false,
		TargetSoC:       70,
		MaxAmps:         16,
		NumPhases:       3,
		SurplusCharging: true,
		MinChargeTime:   15,
		MinSurplus:      2000,
		LowcostCharging: false,
		MaxPrice:        20,
		GridProvider:    "tibber",
		GridStrategy:    1,
		DepartDays:      "12345",
		DepartTime:      "07:00",
		TibberToken:     "",
	}
	GetDB().CreateUpdateVehicle(e)

	if err := GetTeslaAPI().CreateTelemetryConfig(e); err != nil {
		log.Printf("Could not enroll vehicle %d in fleet telemetry: %s\n", e.ID, err.Error())
	}

	SendJSON(w, true)
}

func (router *TeslaRouter) updateVehicle(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	vars := mux.Vars(r)
	vehicleId, _ := strconv.Atoi(vars["id"])

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles(userID)
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
		UserID:          userID,
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
		GridProvider:    m.GridProvider,
		GridStrategy:    m.GridStrategy,
		DepartDays:      m.DepartDays,
		DepartTime:      m.DepartTime,
		TibberToken:     m.TibberToken,
	}
	GetDB().CreateUpdateVehicle(e)

	// If vehicle was not enabled, but is enabled now, update current SoC
	if (eOld != nil) && (e.Enabled) && (!eOld.Enabled) {
		go func() {
			car, err := GetTeslaAPI().InitSession(e, true)
			if err != nil {
				log.Printf("could not init session for vehicle %d on plug in: %s\n", e.ID, err.Error())
				return
			}
			UpdateVehicleDataSaveSoC(e)
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
	userID := GetUserIDFromRequest(r)
	vars := mux.Vars(r)
	vehicleId, _ := strconv.Atoi(vars["id"])

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles(userID)
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

	e := GetDB().GetVehicleByID(vehicleId)
	if e == nil {
		SendBadRequest(w)
		return
	}

	if err := GetTeslaAPI().DeleteTelemetryConfig(e); err != nil {
		log.Printf("Could not remove vehicle %d from fleet telemetry: %s\n", e.ID, err.Error())
	}

	GetDB().DeleteVehicle(vehicle.VehicleID)
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

func (router *TeslaRouter) getVehicleState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleIDString := vars["id"]
	vehicleID, err := strconv.Atoi(vehicleIDString)
	if err != nil {
		SendBadRequest(w)
		return
	}

	userID := GetUserIDFromRequest(r)

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

func (router *TeslaRouter) getLatestSurpluses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleIDString := vars["id"]
	vehicleID, err := strconv.Atoi(vehicleIDString)
	if err != nil {
		SendBadRequest(w)
		return
	}

	userID := GetUserIDFromRequest(r)

	if !GetDB().IsUserOwnerOfVehicle(userID, vehicleID) {
		SendForbidden(w)
		return
	}

	res := GetDB().GetLatestSurplusRecords(vehicleID, 50)
	SendJSON(w, res)
}

func (router *TeslaRouter) getLatestChargingEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleIDString := vars["id"]
	vehicleID, err := strconv.Atoi(vehicleIDString)
	if err != nil {
		log.Println(err)
		SendBadRequest(w)
		return
	}

	userID := GetUserIDFromRequest(r)

	if !GetDB().IsUserOwnerOfVehicle(userID, vehicleID) {
		SendForbidden(w)
		return
	}

	res := GetDB().GetLatestChargingEvents(vehicleID, 50)
	SendJSON(w, res)
}
