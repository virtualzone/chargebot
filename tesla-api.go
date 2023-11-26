package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"
)

type TeslaAPITokenReponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

type TeslaAPIVehicleEntity struct {
	VehicleID   int    `json:"vehicle_id"`
	VIN         string `json:"vin"`
	DisplayName string `json:"display_name"`
}

type TeslaAPIListVehiclesResponse struct {
	Response []TeslaAPIVehicleEntity `json:"response"`
	Count    int                     `json:"count"`
}

func TeslaAPIGetTokens(code string, redirectURI string) (*TeslaAPITokenReponse, error) {
	target := "https://auth.tesla.com/oauth2/v3/token"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", GetConfig().ClientID)
	data.Set("client_secret", GetConfig().ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("audience", GetConfig().Audience)
	r, _ := http.NewRequest("POST", target, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		// TODO
		log.Println(err)
		return nil, err
	}

	var m TeslaAPITokenReponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func TeslaAPIRefreshToken(refreshToken string) (*TeslaAPITokenReponse, error) {
	target := "https://auth.tesla.com/oauth2/v3/token"
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", GetConfig().ClientID)
	data.Set("refresh_token", refreshToken)
	r, _ := http.NewRequest("POST", target, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		// TODO
		log.Println(err)
		return nil, err
	}

	var m TeslaAPITokenReponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func TeslaAPIListVehicles(authToken string) (*TeslaAPIListVehiclesResponse, error) {
	r, _ := http.NewRequest("GET", _configInstance.Audience+"/api/1/vehicles", nil)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var m TeslaAPIListVehiclesResponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
