package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/golang-jwt/jwt/v5"
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

type TeslaAPIBoolResponse struct {
	Result bool   `json:"result"`
	Reason string `json:"reason"`
}

type TeslaAPIListVehiclesResponse struct {
	Response []TeslaAPIVehicleEntity `json:"response"`
	Count    int                     `json:"count"`
}

type TeslaAPIChargeState struct {
	BatteryLevel   int    `json:"battery_level"`
	ChargeAmps     int    `json:"charge_amps"`
	ChargeLimitSoC int    `json:"charge_limit_soc"`
	ChargingState  string `json:"charging_state"`
	Timestamp      int    `json:"timestamp"`
}

type TeslaAPIVehicleData struct {
	VehicleID   int                 `json:"vehicle_id"`
	ChargeState TeslaAPIChargeState `json:"charge_state"`
}

var TeslaAPITokenCache *bigcache.BigCache = nil
var TeslaAPIUserIDToTokenCache *bigcache.BigCache = nil

func TeslaAPIInitTokenCache() {
	config := bigcache.DefaultConfig(8 * time.Hour)
	config.CleanWindow = 1 * time.Minute
	config.HardMaxCacheSize = 1024

	cache, err := bigcache.New(context.Background(), config)
	if err != nil {
		log.Fatalln(err)
	}
	TeslaAPITokenCache = cache

	cache2, err := bigcache.New(context.Background(), config)
	if err != nil {
		log.Fatalln(err)
	}
	TeslaAPIUserIDToTokenCache = cache2
}

func TeslaAPIIsKnownAccessToken(token string) bool {
	v, err := TeslaAPITokenCache.Get(token)
	return err == nil && v != nil
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

	parsedToken, _ := jwt.Parse(m.AccessToken, nil)
	if parsedToken == nil || parsedToken.Claims == nil {
		return nil, errors.New("could not parse jwt")
	}
	sub, _ := parsedToken.Claims.GetSubject()

	// Cache token
	TeslaAPITokenCache.Set(m.AccessToken, []byte("1"))
	TeslaAPIUserIDToTokenCache.Set(sub, []byte(m.AccessToken))

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

	parsedToken, _ := jwt.Parse(m.AccessToken, nil)
	if parsedToken == nil || parsedToken.Claims == nil {
		return nil, errors.New("could not parse jwt")
	}
	sub, _ := parsedToken.Claims.GetSubject()

	// Cache token
	TeslaAPITokenCache.Set(m.AccessToken, []byte("1"))
	TeslaAPIUserIDToTokenCache.Set(sub, []byte(m.AccessToken))

	return &m, nil
}

func TeslaAPIGetOrRefreshAccessToken(userID string) string {
	accessToken := TeslaAPIGetCachedAccessToken(userID)
	if accessToken == "" {
		user := GetUser(userID)
		token, err := TeslaAPIRefreshToken(user.RefreshToken)
		if err != nil {
			log.Println(err)
			return ""
		}
		user.RefreshToken = token.RefreshToken
		CreateUpdateUser(user)
		accessToken = token.AccessToken
	}
	return accessToken
}

func TeslaAPIGetCachedAccessToken(userID string) string {
	token, err := TeslaAPIUserIDToTokenCache.Get(userID)
	if err != nil {
		return ""
	}
	return string(token)
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

func TeslaAPIBoolRequest(authToken string, vehicleID int, cmd string, data string) (bool, error) {
	target := GetConfig().Audience + "/api/1/vehicles/" + strconv.Itoa(vehicleID) + "/command/" + cmd
	r, _ := http.NewRequest("POST", target, strings.NewReader(data))
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Println(err)
		return false, err
	}

	var m TeslaAPIBoolResponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return false, err
	}

	return m.Result, nil
}

func TeslaAPIChargeStart(authToken string, vehicleID int) (bool, error) {
	return TeslaAPIBoolRequest(authToken, vehicleID, "charge_start", `{}`)
}

func TeslaAPIChargeStop(authToken string, vehicleID int) (bool, error) {
	return TeslaAPIBoolRequest(authToken, vehicleID, "charge_stop", `{}`)
}

func TeslaAPISetChargeLimit(authToken string, vehicleID int, limitPercent int) (bool, error) {
	data := `{"percent": "` + strconv.Itoa(limitPercent) + `"}`
	return TeslaAPIBoolRequest(authToken, vehicleID, "set_charge_limit", data)
}

func TeslaAPISetChargeAmps(authToken string, vehicleID int, amps int) (bool, error) {
	data := `{"charging_amps": "` + strconv.Itoa(amps) + `"}`
	return TeslaAPIBoolRequest(authToken, vehicleID, "set_charging_amps", data)
}

func TeslaAPIGetVehicleData(authToken string, vehicleID int) (*TeslaAPIVehicleData, error) {
	target := GetConfig().Audience + "/api/1/vehicles/" + strconv.Itoa(vehicleID) + "/vehicle_data"
	r, _ := http.NewRequest("POST", target, strings.NewReader("{}"))
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var m TeslaAPIVehicleData
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return nil, err
	}

	return &m, nil
}
