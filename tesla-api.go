package main

import (
	"log"
	"net/http"
)

func TeslaAPIListVehicles(authToken string) error {
	r, _ := http.NewRequest("GET", _configInstance.Audience+"/api/1/vehicles", nil)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		// TODO
		log.Println(err)
		return err
	}

	DebugPrintResponseBody(resp.Body)
	return nil
}
