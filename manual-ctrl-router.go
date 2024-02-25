package main

import (
	"math"
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
	vehicle := router.getVehicleFromRequest(r)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	SendJSON(w, true)
	go func() {
		cc := NewChargeController()
		state := GetDB().GetVehicleState(vehicle.ID)
		if state == nil {
			GetDB().SetVehicleStateAmps(vehicle.ID, 0)
			GetDB().SetVehicleStateCharging(vehicle.ID, ChargeStateNotCharging)
			state = GetDB().GetVehicleState(vehicle.ID)
		}
		cc.activateCharging(vehicle, state, int(math.Round((float64)(vehicle.MaxAmps)/2)), ChargeStateChargingOnGrid)
		time.Sleep(30 * time.Second)
		cc.stopCharging(vehicle)
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
