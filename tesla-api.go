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

type TeslaAPI interface {
	InitTokenCache()
	IsKnownAccessToken(token string) bool
	GetTokens(code string, redirectURI string) (*TeslaAPITokenReponse, error)
	RefreshToken(refreshToken string) (*TeslaAPITokenReponse, error)
	GetOrRefreshAccessToken(userID string) string
	GetCachedAccessToken(userID string) string
	ListVehicles(authToken string) ([]TeslaAPIVehicleEntity, error)
	ChargeStart(authToken string, vehicle *Vehicle) (bool, error)
	ChargeStop(authToken string, vehicle *Vehicle) (bool, error)
	SetChargeLimit(authToken string, vehicle *Vehicle, limitPercent int) (bool, error)
	SetChargeAmps(authToken string, vehicle *Vehicle, amps int) (bool, error)
	GetVehicleData(authToken string, vehicle *Vehicle) (*TeslaAPIVehicleData, error)
	WakeUpVehicle(authToken string, vehicle *Vehicle) error
	SetScheduledCharging(authToken string, vehicle *Vehicle, enable bool, minutesAfterMidnight int) (bool, error)
}

type TeslaAPIImpl struct {
	TokenCache         *bigcache.BigCache
	UserIDToTokenCache *bigcache.BigCache
	ReturnPlainInError bool
}

func (a *TeslaAPIImpl) InitTokenCache() {
	config := bigcache.DefaultConfig(8 * time.Hour)
	config.CleanWindow = 1 * time.Minute
	config.HardMaxCacheSize = 1024

	cache, err := bigcache.New(context.Background(), config)
	if err != nil {
		log.Fatalln(err)
	}
	a.TokenCache = cache

	cache2, err := bigcache.New(context.Background(), config)
	if err != nil {
		log.Fatalln(err)
	}
	a.UserIDToTokenCache = cache2
	// a.ReturnPlainInError = true // TODO
}

func (a *TeslaAPIImpl) IsKnownAccessToken(token string) bool {
	v, err := a.TokenCache.Get(token)
	return err == nil && v != nil
}

func (a *TeslaAPIImpl) GetTokens(code string, redirectURI string) (*TeslaAPITokenReponse, error) {
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

	resp, err := RetryHTTPRequest(r)
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
	a.TokenCache.Set(m.AccessToken, []byte("1"))
	a.UserIDToTokenCache.Set(sub, []byte(m.AccessToken))

	return &m, nil
}

func (a *TeslaAPIImpl) RefreshToken(refreshToken string) (*TeslaAPITokenReponse, error) {
	target := "https://auth.tesla.com/oauth2/v3/token"
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", GetConfig().ClientID)
	data.Set("refresh_token", refreshToken)
	r, _ := http.NewRequest("POST", target, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := RetryHTTPRequest(r)
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
	a.TokenCache.Set(m.AccessToken, []byte("1"))
	a.UserIDToTokenCache.Set(sub, []byte(m.AccessToken))

	return &m, nil
}

func (a *TeslaAPIImpl) GetOrRefreshAccessToken(userID string) string {
	accessToken := a.GetCachedAccessToken(userID)
	if accessToken == "" {
		user := GetDB().GetUser(userID)
		token, err := a.RefreshToken(user.RefreshToken)
		if err != nil {
			log.Println(err)
			return ""
		}
		user.RefreshToken = token.RefreshToken
		GetDB().CreateUpdateUser(user)
		accessToken = token.AccessToken
	}
	return accessToken
}

func (a *TeslaAPIImpl) GetCachedAccessToken(userID string) string {
	token, err := a.UserIDToTokenCache.Get(userID)
	if err != nil {
		return ""
	}
	return string(token)
}

func (a *TeslaAPIImpl) ListVehicles(authToken string) ([]TeslaAPIVehicleEntity, error) {
	r, _ := http.NewRequest("GET", _configInstance.Audience+"/api/1/vehicles", nil)

	resp, err := RetryHTTPJSONRequest(r, authToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var m TeslaAPIListVehiclesResponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return nil, err
	}

	if m.Error != "" {
		return nil, fmt.Errorf("api response error: %s (%s), http status %d", m.Error, m.ErrorDescription, resp.StatusCode)
	}

	return m.Response, nil
}

func (a *TeslaAPIImpl) boolRequest(authToken string, vehicle *Vehicle, cmd string, data string) (bool, error) {
	target := GetConfig().Audience + "/api/1/vehicles/" + vehicle.VIN + "/command/" + cmd
	log.Printf("Sending request to %s: %s\n", target, data)
	r, _ := http.NewRequest("POST", target, strings.NewReader(data))

	resp, err := RetryHTTPJSONRequest(r, authToken)
	if err != nil {
		log.Println(err)
		return false, err
	}

	if a.ReturnPlainInError {
		body, _ := DebugGetResponseBody(r.Body)
		return false, errors.New(fmt.Sprintf("http status %d, body: %s", resp.StatusCode, body))
	}

	var m TeslaAPIBoolResponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return false, err
	}

	if m.Error != "" {
		return false, fmt.Errorf("api response error: %s (%s), http status %d", m.Error, m.ErrorDescription, resp.StatusCode)
	}

	return m.Response.Result, nil
}

func (a *TeslaAPIImpl) ChargeStart(authToken string, vehicle *Vehicle) (bool, error) {
	return a.boolRequest(authToken, vehicle, "charge_start", `{}`)
}

func (a *TeslaAPIImpl) ChargeStop(authToken string, vehicle *Vehicle) (bool, error) {
	return a.boolRequest(authToken, vehicle, "charge_stop", `{}`)
}

func (a *TeslaAPIImpl) SetChargeLimit(authToken string, vehicle *Vehicle, limitPercent int) (bool, error) {
	data := `{"percent": "` + strconv.Itoa(limitPercent) + `"}`
	return a.boolRequest(authToken, vehicle, "set_charge_limit", data)
}

func (a *TeslaAPIImpl) SetChargeAmps(authToken string, vehicle *Vehicle, amps int) (bool, error) {
	data := `{"charging_amps": ` + strconv.Itoa(amps) + `}`
	return a.boolRequest(authToken, vehicle, "set_charging_amps", data)
}

func (a *TeslaAPIImpl) GetVehicleData(authToken string, vehicle *Vehicle) (*TeslaAPIVehicleData, error) {
	target := GetConfig().Audience + "/api/1/vehicles/" + vehicle.VIN + "/vehicle_data"
	r, _ := http.NewRequest("GET", target, nil)

	resp, err := RetryHTTPJSONRequest(r, authToken)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var m TeslaAPIVehicleDataResponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return nil, err
	}

	if m.Error != "" {
		return nil, fmt.Errorf("api response error: %s (%s), http status %d", m.Error, m.ErrorDescription, resp.StatusCode)
	}

	return &m.Response, nil
}

func (a *TeslaAPIImpl) WakeUpVehicle(authToken string, vehicle *Vehicle) error {
	target := GetConfig().Audience + "/api/1/vehicles/" + vehicle.VIN + "/wake_up"
	r, _ := http.NewRequest("POST", target, strings.NewReader("{}"))

	resp, err := RetryHTTPJSONRequest(r, authToken)
	if err != nil {
		log.Println(err)
		return err
	}

	if a.ReturnPlainInError {
		body, _ := DebugGetResponseBody(r.Body)
		return errors.New(fmt.Sprintf("http status %d, body: %s", resp.StatusCode, body))
	}

	var m TeslaAPIErrorResponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return err
	}

	if m.Error != "" {
		return fmt.Errorf("api response error: %s (%s), http status %d", m.Error, m.ErrorDescription, resp.StatusCode)
	}

	return nil
}

func (a *TeslaAPIImpl) SetScheduledCharging(authToken string, vehicle *Vehicle, enable bool, minutesAfterMidnight int) (bool, error) {
	payload := `{"enable": ` + strconv.FormatBool(enable) + `, "time": ` + strconv.Itoa(minutesAfterMidnight) + `}`
	return a.boolRequest(authToken, vehicle, "set_scheduled_charging", payload)
}
