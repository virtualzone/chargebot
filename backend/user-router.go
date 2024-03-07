package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type UserRouter struct {
	WebsocketUpgrader websocket.Upgrader
}

type AuthenticatedUserRequest struct {
	Vehicle *Vehicle
	UserID  string
}

type PasswordProtectedRequest struct {
	Password string `json:"password"`
}

type SurplusRecordingRequest struct {
	PasswordProtectedRequest
	SurplusWatts        int `json:"surplus_watts"`
	InverterActivePower int `json:"inverter_active_power_watts"`
	Consumption         int `json:"consumption_watts"`
}

type SurplusRecordingResponse struct {
}

type PlugInOutRequest struct {
	Password string `json:"password"`
}

func (router *UserRouter) SetupRoutes(s *mux.Router) {
	router.WebsocketUpgrader = websocket.Upgrader{}
	s.HandleFunc("/{token}/ws", router.websocket).Methods("GET")
	s.HandleFunc("/{token}/surplus", router.recordSurplus).Methods("POST")
	s.HandleFunc("/{token}/{vin}/plugged_in", router.vehiclePluggedIn).Methods("POST")
	s.HandleFunc("/{token}/{vin}/unplugged", router.vehicleUnplugged).Methods("POST")
	s.HandleFunc("/{token}/list_vehicles", router.listVehicles).Methods("POST")
	/*
		s.HandleFunc("/{token}/{vin}/init_session", router.initSession).Methods("POST")
		s.HandleFunc("/{token}/{vin}/charge_start", router.chargeStart).Methods("POST")
		s.HandleFunc("/{token}/{vin}/charge_stop", router.chargeStop).Methods("POST")
		s.HandleFunc("/{token}/{vin}/set_charge_limit", router.setChargeLimit).Methods("POST")
		s.HandleFunc("/{token}/{vin}/set_charge_amps", router.setChargeAmps).Methods("POST")
		s.HandleFunc("/{token}/{vin}/vehicle_data", router.vehicleData).Methods("POST")
	*/
}

func (router *UserRouter) websocket(w http.ResponseWriter, r *http.Request) {
	c, err := router.WebsocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		// TODO
		log.Printf("recv: %s, type: %v\n", message, mt)
		err = c.WriteMessage(mt, []byte("hey, thanks for sending me "+string(message)))
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func (router *UserRouter) recordSurplus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	var m *SurplusRecordingRequest
	if err := UnmarshalBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}

	if !GetDB().IsTokenPasswordValid(token, m.Password) {
		SendUnauthorized(w)
		return
	}

	userID := GetDB().GetAPITokenUserID(token)
	if userID == "" {
		SendBadRequest(w)
		return
	}

	surplus := m.SurplusWatts
	if surplus == 0 {
		if m.InverterActivePower != 0 || m.Consumption != 0 {
			surplus = m.InverterActivePower - m.Consumption
		}
	}

	GetDB().RecordSurplus(userID, surplus)
	SendJSON(w, true)
}

func (router *UserRouter) updateVehiclePlugState(w http.ResponseWriter, r *http.Request, pluggedIn bool) {
	vars := mux.Vars(r)
	token := vars["token"]
	vin := vars["vin"]

	var m *PasswordProtectedRequest
	if err := UnmarshalBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}

	if !GetDB().IsTokenPasswordValid(token, m.Password) {
		SendUnauthorized(w)
		return
	}

	userID := GetDB().GetAPITokenUserID(token)
	if userID == "" {
		SendBadRequest(w)
		return
	}

	if !GetDB().IsUserOwnerOfVehicle(userID, vin) {
		SendForbidden(w)
		return
	}

	vehicle := GetDB().GetVehicleByVIN(vin)
	if vehicle == nil {
		SendNotFound(w)
		return
	}

	if pluggedIn {
		OnVehiclePluggedIn(vehicle)
	} else {
		state := GetDB().GetVehicleState(vehicle.VIN)
		OnVehicleUnplugged(vehicle, state)
	}
}

func (router *UserRouter) vehiclePluggedIn(w http.ResponseWriter, r *http.Request) {
	router.updateVehiclePlugState(w, r, true)
}

func (router *UserRouter) vehicleUnplugged(w http.ResponseWriter, r *http.Request) {
	router.updateVehiclePlugState(w, r, false)
}

func (router *UserRouter) listVehicles(w http.ResponseWriter, r *http.Request) {
	a := router.authenticateUserRequest(w, r)
	if a == nil {
		return
	}

	GetTeslaAPI().ListVehicles(a.UserID)
}

func (router *UserRouter) authenticateUserRequest(w http.ResponseWriter, r *http.Request) *AuthenticatedUserRequest {
	vars := mux.Vars(r)
	token := vars["token"]
	vin := vars["vin"]

	var m *PasswordProtectedRequest
	if err := UnmarshalBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return nil
	}

	if !GetDB().IsTokenPasswordValid(token, m.Password) {
		SendUnauthorized(w)
		return nil
	}

	userID := GetDB().GetAPITokenUserID(token)
	if userID == "" {
		SendBadRequest(w)
		return nil
	}

	var vehicle *Vehicle = nil
	if vin != "" {
		if !GetDB().IsUserOwnerOfVehicle(userID, vin) {
			SendForbidden(w)
			return nil
		}

		vehicle = GetDB().GetVehicleByVIN(vin)
		if vehicle == nil {
			SendNotFound(w)
			return nil
		}
	}

	return &AuthenticatedUserRequest{
		Vehicle: vehicle,
		UserID:  userID,
	}
}
