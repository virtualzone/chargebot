package main

import (
	"net/http"
)

func TeslaAPIListVehicles(authToken string) {
	r, _ := http.NewRequest("GET", _configInstance.Audience+"/api/1/vehicles", nil)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+authToken)
}
