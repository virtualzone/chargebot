package main

import (
	"net/http"

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
	s.HandleFunc("/my_vehicles", router.myVehicles).Methods("GET")
	s.HandleFunc("/api_token_create", router.createAPIToken).Methods("POST")
	s.HandleFunc("/api_token_update/{id}", router.updateAPIToken).Methods("POST")
}

func (router *TeslaRouter) myVehicles(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	list := GetDB().GetVehicles(userID)
	SendJSON(w, list)
}

func (router *TeslaRouter) createAPIToken(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)

	password := GeneratePassword(16, true, false)
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

	password := GeneratePassword(16, true, false)
	GetDB().UpdateAPITokenPassword(token, password)

	resp := GetAPITokenResponse{
		Token:    token,
		UserID:   userID,
		Password: password,
	}
	SendJSON(w, resp)
}
