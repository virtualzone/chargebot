package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

type TeslaRouter struct {
}

func (router *TeslaRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/vehicles", router.listVehicles).Methods("GET")
}

func (router *TeslaRouter) listVehicles(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	err := TeslaAPIListVehicles(authToken)
	if err != nil {
		SendInternalServerError(w)
		return
	}
}
