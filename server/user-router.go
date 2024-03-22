package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
	. "github.com/virtualzone/chargebot/goshared"
)

type UserRouter struct {
	WebsocketUpgrader websocket.Upgrader
	VINToSessionCache *InMemoryCache
}

type AuthenticatedUserRequest struct {
	Vehicle *Vehicle
	UserID  string
}

func (router *UserRouter) SetupRoutes(s *mux.Router) {
	router.VINToSessionCache = NewInMemoryCache(5 * time.Minute)
	router.WebsocketUpgrader = websocket.Upgrader{}
	s.HandleFunc("/{token}/ping", router.ping).Methods("POST")
	s.HandleFunc("/{token}/ws", router.websocket).Methods("GET")
	s.HandleFunc("/{token}/list_vehicles", router.listVehicles).Methods("POST")
	s.HandleFunc("/{token}/vehicle_add/{vin}", router.addVehicle).Methods("POST")
	s.HandleFunc("/{token}/vehicle_delete/{vin}", router.deleteVehicle).Methods("POST")
	s.HandleFunc("/{token}/{vin}/state", router.getTelemetryState).Methods("POST")
	s.HandleFunc("/{token}/{vin}/charge_start", router.chargeStart).Methods("POST")
	s.HandleFunc("/{token}/{vin}/charge_stop", router.chargeStop).Methods("POST")
	s.HandleFunc("/{token}/{vin}/set_charge_limit", router.setChargeLimit).Methods("POST")
	s.HandleFunc("/{token}/{vin}/set_charge_amps", router.setChargeAmps).Methods("POST")
	s.HandleFunc("/{token}/{vin}/vehicle_data", router.vehicleData).Methods("POST")
	s.HandleFunc("/{token}/{vin}/wakeup", router.wakeup).Methods("POST")
	s.HandleFunc("/{token}/{vin}/create_telemetry_config", router.createTelemetryConfig).Methods("POST")
	s.HandleFunc("/{token}/{vin}/delete_telemetry_config", router.deleteTelemetryConfig).Methods("POST")
}

func (router *UserRouter) getOrInitSession(accessToken, vin string) (*vehicle.Vehicle, error) {
	session := router.VINToSessionCache.Get(vin)
	if session != nil {
		return session.(*vehicle.Vehicle), nil
	}
	car, err := GetTeslaAPI().InitSession(accessToken, vin)
	if err != nil {
		return nil, err
	}
	router.VINToSessionCache.Set(vin, car)
	return car, nil
}

func (router *UserRouter) sendError(w http.ResponseWriter, err error) {
	res := ErrorResponse{
		Error: err.Error(),
	}
	w.WriteHeader(http.StatusInternalServerError)
	SendJSON(w, res)
}

func (router *UserRouter) sendWebsocketState(c *websocket.Conn, state *PersistedTelemetryState) {
	json, _ := json.Marshal(state)
	c.WriteMessage(websocket.TextMessage, json)
}

func (router *UserRouter) websocket(w http.ResponseWriter, r *http.Request) {
	c, err := router.WebsocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	vars := mux.Vars(r)
	token := vars["token"]
	authorized := false
	userID := ""
	lastTs := make(map[string]int64)
	vehicleVINs := []string{}

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	defer c.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}

			if mt != websocket.TextMessage {
				continue
			}

			text := string(message)
			if strings.Index(text, "{\"password\":") == 0 {
				var m PasswordProtectedRequest
				if err = json.Unmarshal(message, &m); err != nil {
					c.WriteMessage(mt, []byte("false"))
					continue
				}
				if !GetDB().IsTokenPasswordValid(token, m.Password) {
					c.WriteMessage(mt, []byte("false"))
					continue
				}
				authorized = true
				c.WriteMessage(mt, []byte("true"))

				userID = GetDB().GetAPITokenUserID(token)
				vehicles := GetDB().GetVehicles(userID)

				for _, vehicle := range vehicles {
					vehicleVINs = append(vehicleVINs, vehicle.VIN)
					GetTelemetryQueue().AddActiveVIN(vehicle.VIN)
					defer GetTelemetryQueue().RemoveActiveVIN(vehicle.VIN)
					state := GetDB().GetTelemetryState(vehicle.VIN)
					if state != nil {
						router.sendWebsocketState(c, state)
						lastTs[vehicle.VIN] = state.UTC
					}
				}

				continue
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if authorized {
				for _, vin := range vehicleVINs {
					state := GetTelemetryQueue().GetState(vin)
					if state != nil {
						before, ok := lastTs[vin]
						if !ok || before < state.UTC {
							router.sendWebsocketState(c, state)
							lastTs[vin] = state.UTC
						}
					}
				}
			}
		}
	}
}

func (router *UserRouter) ping(w http.ResponseWriter, r *http.Request) {
	var m PasswordProtectedRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m, false)
	if a == nil {
		return
	}
	SendJSON(w, true)
}

func (router *UserRouter) getTelemetryState(w http.ResponseWriter, r *http.Request) {
	var m PasswordProtectedRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m, true)
	if a == nil {
		return
	}

	state := GetDB().GetTelemetryState(a.Vehicle.VIN)
	SendJSON(w, state)
}

func (router *UserRouter) listVehicles(w http.ResponseWriter, r *http.Request) {
	var m AccessTokenRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m.PasswordProtectedRequest, false)
	if a == nil {
		return
	}

	res, err := GetTeslaAPI().ListVehicles(m.AccessToken)
	if err != nil {
		router.sendError(w, err)
		return
	}
	SendJSON(w, res)
}

func (router *UserRouter) addVehicle(w http.ResponseWriter, r *http.Request) {
	var m AccessTokenRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}

	vars := mux.Vars(r)
	token := vars["token"]
	vin := vars["vin"]

	if !GetDB().IsTokenPasswordValid(token, m.Password) {
		SendUnauthorized(w)
		return
	}

	userID := GetDB().GetAPITokenUserID(token)
	if userID == "" {
		SendBadRequest(w)
		return
	}

	res, err := GetTeslaAPI().ListVehicles(m.AccessToken)
	if err != nil {
		router.sendError(w, err)
		return
	}

	isOwner := false
	for _, vehicle := range res {
		if vehicle.VIN == vin {
			isOwner = true
		}
	}

	if !isOwner {
		SendForbidden(w)
		return
	}

	e := &Vehicle{
		VIN:    vin,
		UserID: userID,
	}
	GetDB().CreateUpdateVehicle(e)
	SendJSON(w, true)
}

func (router *UserRouter) deleteVehicle(w http.ResponseWriter, r *http.Request) {
	var m AccessTokenRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}

	vars := mux.Vars(r)
	token := vars["token"]
	vin := vars["vin"]

	if !GetDB().IsTokenPasswordValid(token, m.Password) {
		SendUnauthorized(w)
		return
	}

	userID := GetDB().GetAPITokenUserID(token)
	if userID == "" {
		SendBadRequest(w)
		return
	}

	res, err := GetTeslaAPI().ListVehicles(m.AccessToken)
	if err != nil {
		router.sendError(w, err)
		return
	}

	isOwner := false
	for _, vehicle := range res {
		if vehicle.VIN == vin {
			isOwner = true
		}
	}

	if !isOwner {
		SendForbidden(w)
		return
	}

	GetDB().DeleteVehicle(vin)
	SendJSON(w, true)
}

func (router *UserRouter) chargeStart(w http.ResponseWriter, r *http.Request) {
	var m AccessTokenRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m.PasswordProtectedRequest, true)
	if a == nil {
		return
	}

	car, err := router.getOrInitSession(m.AccessToken, a.Vehicle.VIN)
	if err != nil {
		router.sendError(w, err)
		return
	}
	if err := GetTeslaAPI().ChargeStart(car); err != nil {
		router.sendError(w, err)
		return
	}
	SendJSON(w, true)
}

func (router *UserRouter) chargeStop(w http.ResponseWriter, r *http.Request) {
	var m AccessTokenRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m.PasswordProtectedRequest, true)
	if a == nil {
		return
	}

	car, err := router.getOrInitSession(m.AccessToken, a.Vehicle.VIN)
	if err != nil {
		router.sendError(w, err)
		return
	}
	if err := GetTeslaAPI().ChargeStop(car); err != nil {
		router.sendError(w, err)
		return
	}
	SendJSON(w, true)
}

func (router *UserRouter) vehicleData(w http.ResponseWriter, r *http.Request) {
	var m AccessTokenRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m.PasswordProtectedRequest, true)
	if a == nil {
		return
	}

	res, err := GetTeslaAPI().GetVehicleData(m.AccessToken, a.Vehicle.VIN)
	if err != nil {
		router.sendError(w, err)
		return
	}
	SendJSON(w, res)
}

func (router *UserRouter) setChargeLimit(w http.ResponseWriter, r *http.Request) {
	var m SetChargeLimitRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m.PasswordProtectedRequest, true)
	if a == nil {
		return
	}

	car, err := router.getOrInitSession(m.AccessToken, a.Vehicle.VIN)
	if err != nil {
		router.sendError(w, err)
		return
	}
	if err := GetTeslaAPI().SetChargeLimit(car, m.ChargeLimit); err != nil {
		router.sendError(w, err)
		return
	}
	SendJSON(w, true)
}

func (router *UserRouter) setChargeAmps(w http.ResponseWriter, r *http.Request) {
	var m SetChargeAmpsRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m.PasswordProtectedRequest, true)
	if a == nil {
		return
	}

	car, err := router.getOrInitSession(m.AccessToken, a.Vehicle.VIN)
	if err != nil {
		router.sendError(w, err)
		return
	}
	if err := GetTeslaAPI().SetChargeAmps(car, m.Amps); err != nil {
		router.sendError(w, err)
		return
	}
	SendJSON(w, true)
}

func (router *UserRouter) wakeup(w http.ResponseWriter, r *http.Request) {
	var m AccessTokenRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m.PasswordProtectedRequest, true)
	if a == nil {
		return
	}

	err := GetTeslaAPI().Wakeup(m.AccessToken, a.Vehicle.VIN)
	if err != nil {
		router.sendError(w, err)
		return
	}

	SendJSON(w, true)
}

func (router *UserRouter) createTelemetryConfig(w http.ResponseWriter, r *http.Request) {
	var m AccessTokenRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m.PasswordProtectedRequest, true)
	if a == nil {
		return
	}

	err := GetTeslaAPI().CreateTelemetryConfig(m.AccessToken, a.Vehicle.VIN)
	if err != nil {
		router.sendError(w, err)
		return
	}

	SendJSON(w, true)
}

func (router *UserRouter) deleteTelemetryConfig(w http.ResponseWriter, r *http.Request) {
	var m AccessTokenRequest
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	a := router.authenticateUserRequest(w, r, &m.PasswordProtectedRequest, true)
	if a == nil {
		return
	}

	err := GetTeslaAPI().DeleteTelemetryConfig(m.AccessToken, a.Vehicle.VIN)
	if err != nil {
		router.sendError(w, err)
		return
	}

	SendJSON(w, true)
}

func (router *UserRouter) authenticateUserRequest(w http.ResponseWriter, r *http.Request, m *PasswordProtectedRequest, requireVehicle bool) *AuthenticatedUserRequest {
	vars := mux.Vars(r)
	token := vars["token"]
	vin := vars["vin"]

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
	if requireVehicle && vehicle == nil {
		SendBadRequest(w)
		return nil
	}

	return &AuthenticatedUserRequest{
		Vehicle: vehicle,
		UserID:  userID,
	}
}
