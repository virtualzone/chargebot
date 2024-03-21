package main

import (
	"log"
	"net/http"
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

type ErrorResponse struct {
	Error string `json:"error"`
}

func (router *UserRouter) SetupRoutes(s *mux.Router) {
	router.VINToSessionCache = NewInMemoryCache(5 * time.Minute)
	router.WebsocketUpgrader = websocket.Upgrader{}
	s.HandleFunc("/{token}/ws", router.websocket).Methods("GET")
	s.HandleFunc("/{token}/list_vehicles", router.listVehicles).Methods("POST")
	s.HandleFunc("/{token}/{vin}/charge_start", router.chargeStart).Methods("POST")
	s.HandleFunc("/{token}/{vin}/charge_stop", router.chargeStop).Methods("POST")
	s.HandleFunc("/{token}/{vin}/set_charge_limit", router.setChargeLimit).Methods("POST")
	s.HandleFunc("/{token}/{vin}/set_charge_amps", router.setChargeAmps).Methods("POST")
	s.HandleFunc("/{token}/{vin}/vehicle_data", router.vehicleData).Methods("POST")
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

func (router *UserRouter) listVehicles(w http.ResponseWriter, r *http.Request) {
	var m *AccessTokenRequest
	a := router.authenticateUserRequest(w, r, &m, false)
	if a == nil {
		return
	}

	res, err := GetTeslaAPI().ListVehicles(m.AccessToken)
	if err != nil {
		SendInternalServerError(w)
		return
	}
	SendJSON(w, res)
}

func (router *UserRouter) chargeStart(w http.ResponseWriter, r *http.Request) {
	var m *AccessTokenRequest
	a := router.authenticateUserRequest(w, r, &m, true)
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
	var m *AccessTokenRequest
	a := router.authenticateUserRequest(w, r, &m, true)
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
	var m *AccessTokenRequest
	a := router.authenticateUserRequest(w, r, &m, true)
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
	var m *SetChargeLimitRequest
	a := router.authenticateUserRequest(w, r, &m, true)
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
	var m *SetChargeAmpsRequest
	a := router.authenticateUserRequest(w, r, &m, true)
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

func (router *UserRouter) authenticateUserRequest(w http.ResponseWriter, r *http.Request, o interface{}, requireVehicle bool) *AuthenticatedUserRequest {
	vars := mux.Vars(r)
	token := vars["token"]
	vin := vars["vin"]

	if err := UnmarshalBody(r.Body, &o); err != nil {
		SendBadRequest(w)
		return nil
	}

	m, ok := o.(PasswordProtectedRequest)
	if !ok {
		log.Println("authenticateUserRequest() failed with interface not being of type PasswordProtectedRequest")
		SendInternalServerError(w)
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
	if requireVehicle && vehicle == nil {
		SendBadRequest(w)
		return nil
	}

	return &AuthenticatedUserRequest{
		Vehicle: vehicle,
		UserID:  userID,
	}
}
