package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type ManualControlRouter struct {
}

type ManualControlResponse struct {
	Error string `json:"err"`
}

func (router *ManualControlRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/{id}/testDrive", router.testDrive).Methods("POST")
}

func (router *ManualControlRouter) testDrive(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	SendJSON(w, true)
	go func() {
		cc := NewChargeController()
		state := GetDB().GetVehicleState(vehicle.ID)
		cc.activateCharging(authToken, vehicle, state, vehicle.MaxAmps, ChargeStateChargingOnGrid)
		time.Sleep(30 * time.Second)
		cc.stopCharging(authToken, vehicle)
	}()
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
