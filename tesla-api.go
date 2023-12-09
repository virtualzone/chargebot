package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/golang-jwt/jwt/v5"
)

type TeslaAPIErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type TeslaAPITokenReponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

type TeslaAPIVehicleEntity struct {
	VehicleID   int    `json:"vehicle_id"`
	VIN         string `json:"vin"`
	DisplayName string `json:"display_name"`
}

type TeslaAPIBool struct {
	Result bool   `json:"result"`
	Reason string `json:"reason"`
}

type TeslaAPIBoolResponse struct {
	TeslaAPIErrorResponse
	Response TeslaAPIBool `json:"response"`
}

type TeslaAPIListVehiclesResponse struct {
	TeslaAPIErrorResponse
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

type TeslaAPIVehicleDataResponse struct {
	TeslaAPIErrorResponse
	Response TeslaAPIVehicleData `json:"response"`
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

func TeslaAPIListVehicles(authToken string) ([]TeslaAPIVehicleEntity, error) {
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

	if m.Error != "" {
		return nil, fmt.Errorf("api response error: %s (%s)", m.Error, m.ErrorDescription)
	}

	return m.Response, nil
}

func TeslaAPIBoolRequest(authToken string, vehicle *Vehicle, cmd string, data string) (bool, error) {
	target := GetConfig().Audience + "/api/1/vehicles/" + vehicle.VIN + "/command/" + cmd
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

	if m.Error != "" {
		return false, fmt.Errorf("api response error: %s (%s)", m.Error, m.ErrorDescription)
	}

	return m.Response.Result, nil
}

func TeslaAPIChargeStart(authToken string, vehicle *Vehicle) (bool, error) {
	return TeslaAPIBoolRequest(authToken, vehicle, "charge_start", `{}`)
}

func TeslaAPIChargeStop(authToken string, vehicle *Vehicle) (bool, error) {
	return TeslaAPIBoolRequest(authToken, vehicle, "charge_stop", `{}`)
}

func TeslaAPISetChargeLimit(authToken string, vehicle *Vehicle, limitPercent int) (bool, error) {
	data := `{"percent": "` + strconv.Itoa(limitPercent) + `"}`
	return TeslaAPIBoolRequest(authToken, vehicle, "set_charge_limit", data)
}

func TeslaAPISetChargeAmps(authToken string, vehicle *Vehicle, amps int) (bool, error) {
	data := `{"charging_amps": "` + strconv.Itoa(amps) + `"}`
	return TeslaAPIBoolRequest(authToken, vehicle, "set_charging_amps", data)
}

func TeslaAPIGetVehicleData(authToken string, vehicle *Vehicle) (*TeslaAPIVehicleData, error) {
	target := GetConfig().Audience + "/api/1/vehicles/" + vehicle.VIN + "/vehicle_data"
	r, _ := http.NewRequest("GET", target, nil)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var m TeslaAPIVehicleDataResponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return nil, err
	}

	if m.Error != "" {
		return nil, fmt.Errorf("api response error: %s (%s)", m.Error, m.ErrorDescription)
	}

	return &m.Response, nil
}
