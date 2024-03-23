package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	. "github.com/virtualzone/chargebot/goshared"
)

type TeslaRouter struct {
}

func (router *TeslaRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/vehicles", router.listVehicles).Methods("GET")
	s.HandleFunc("/my_vehicles", router.myVehicles).Methods("GET")
	s.HandleFunc("/vehicle_add/{vin}", router.addVehicle).Methods("POST")
	s.HandleFunc("/vehicle_update/{vin}", router.updateVehicle).Methods("PUT")
	s.HandleFunc("/vehicle_delete/{vin}", router.deleteVehicle).Methods("DELETE")
	s.HandleFunc("/state/{vin}", router.getVehicleState).Methods("GET")
	s.HandleFunc("/surplus", router.getLatestSurpluses).Methods("GET")
	s.HandleFunc("/events/{vin}", router.getLatestChargingEvents).Methods("GET")
	s.HandleFunc("/permanent_error", router.getPermanentError).Methods("GET")
	s.HandleFunc("/resolve_permanent_error", router.resolvePermanentError).Methods("POST")
}

func (router *TeslaRouter) listVehicles(w http.ResponseWriter, r *http.Request) {
	list, err := GetTeslaAPI().ListVehicles()
	if err != nil {
		log.Println(err)
		SendUnauthorized(w)
		return
	}
	SendJSON(w, list)
}

func (router *TeslaRouter) myVehicles(w http.ResponseWriter, r *http.Request) {
	list := GetDB().GetVehicles()
	SendJSON(w, list)
}

func (router *TeslaRouter) addVehicle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vin := vars["vin"]

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles()
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

	if err := GetTeslaAPI().RegisterVehicle(vehicle.VIN); err != nil {
		SendInternalServerError(w)
		return
	}

	e := &Vehicle{
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

	if err := GetTeslaAPI().CreateTelemetryConfig(e.VIN); err != nil {
		log.Printf("Could not enroll vehicle %s in fleet telemetry: %s\n", e.VIN, err.Error())
	}

	// Reconnect websocket so server re-caches the correct vins
	GetTelemetryPoller().Reconnect()

	SendJSON(w, true)
}

func (router *TeslaRouter) updateVehicle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vin := vars["vin"]

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles()
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

	var m *Vehicle
	UnmarshalValidateBody(r.Body, &m)

	//eOld := GetDB().GetVehicleByVIN(vehicle.VIN)

	e := &Vehicle{
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
	/*
		if (eOld != nil) && (e.Enabled) && (!eOld.Enabled) {
			state := GetDB().GetVehicleState(e.VIN)
			go func() {
				car, err := GetTeslaAPI().InitSession(e, true)
				if err != nil {
					log.Printf("could not init session for vehicle %s on plug in: %s\n", e.VIN, err.Error())
					return
				}
				time.Sleep(5 * time.Second)
				if err := GetTeslaAPI().ChargeStop(car); err != nil {
					log.Printf("could not stop charging for vehicle %s on plug in: %s\n", e.VIN, err.Error())
				}
			}()
		}
	*/

	SendJSON(w, true)
}

func (router *TeslaRouter) deleteVehicle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vin := vars["vin"]

	// Check if vehicle belongs to request user
	list, err := GetTeslaAPI().ListVehicles()
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

	if err := GetTeslaAPI().DeleteTelemetryConfig(e.VIN); err != nil {
		log.Printf("Could not remove vehicle %s from fleet telemetry: %s\n", e.VIN, err.Error())
	}

	GetDB().DeleteVehicle(vehicle.VIN)

	GetTeslaAPI().UnregisterVehicle(vehicle.VIN)

	// Reconnect websocket so server re-caches the correct vins
	GetTelemetryPoller().Reconnect()

	SendJSON(w, true)
}

func (router *TeslaRouter) getVehicleState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vin := vars["vin"]

	state := GetDB().GetVehicleState(vin)
	if state == nil {
		SendNotFound(w)
		return
	}
	SendJSON(w, state)
}

func (router *TeslaRouter) getLatestSurpluses(w http.ResponseWriter, r *http.Request) {
	res := GetDB().GetLatestSurplusRecords(50)
	SendJSON(w, res)
}

func (router *TeslaRouter) getLatestChargingEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vin := vars["vin"]

	res := GetDB().GetLatestChargingEvents(vin, 50)
	SendJSON(w, res)
}

func (router *TeslaRouter) getPermanentError(w http.ResponseWriter, r *http.Request) {
	val := GetDB().GetSetting(SettingsPermanentError)
	SendJSON(w, val == "1")
}

func (router *TeslaRouter) resolvePermanentError(w http.ResponseWriter, r *http.Request) {
	GetDB().SetSetting(SettingsPermanentError, "")
	SendJSON(w, true)
}
