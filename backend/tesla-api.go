package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/teslamotors/vehicle-command/pkg/account"
	"github.com/teslamotors/vehicle-command/pkg/protocol/protobuf/universalmessage"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
	. "github.com/virtualzone/chargebot/goshared"
)

type TeslaAPIErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type TeslaAPIBool struct {
	Result bool   `json:"result"`
	Reason string `json:"reason"`
}

type TeslaAPIBoolResponse struct {
	TeslaAPIErrorResponse
	Response TeslaAPIBool `json:"response"`
}

type TeslaAPIVehicleUpdateResponse struct {
	TeslaAPIErrorResponse
	NumUpdatedVehicles int `json:"updated_vehicles"`
}

type TeslaAPIListVehiclesResponse struct {
	TeslaAPIErrorResponse
	Response []TeslaAPIVehicleEntity `json:"response"`
	Count    int                     `json:"count"`
}

type TeslaAPIVehicleDataResponse struct {
	TeslaAPIErrorResponse
	Response TeslaAPIVehicleData `json:"response"`
}

type TeslaAPITelemetryField struct {
	IntervalSeconds int `json:"interval_seconds"`
}

type TeslaAPITelemetryConfig struct {
	Hostname   string                            `json:"hostname"`
	CA         string                            `json:"ca"`
	Fields     map[string]TeslaAPITelemetryField `json:"fields"`
	AlertTypes []string                          `json:"alert_types"`
	Expiration int64                             `json:"exp"`
	Port       int                               `json:"port"`
}

type TeslaAPITelemetryConfigCreate struct {
	VINs   []string                `json:"vins"`
	Config TeslaAPITelemetryConfig `json:"config"`
}

type TeslaAPI interface {
	GetTokens(userID string, code string, redirectURI string) (*TeslaAPITokenReponse, error)
	RefreshToken(userID string, refreshToken string) (*TeslaAPITokenReponse, error)
	InitSession(accessToken string, vin string) (*vehicle.Vehicle, error)
	ListVehicles(accessToken string) ([]TeslaAPIVehicleEntity, error)
	ChargeStart(car *vehicle.Vehicle) error
	ChargeStop(car *vehicle.Vehicle) error
	SetChargeLimit(car *vehicle.Vehicle, limitPercent int) error
	SetChargeAmps(car *vehicle.Vehicle, amps int) error
	GetVehicleData(accessToken string, vin string) (*TeslaAPIVehicleData, error)
	Wakeup(accessToken string, vin string) error
	CreateTelemetryConfig(accessToken string, vin string) error
	DeleteTelemetryConfig(accessToken string, vin string) error
}

type TeslaAPIImpl struct {
	//UserIDToTokenCache *bigcache.BigCache
}

/*
func (a *TeslaAPIImpl) InitTokenCache() {
	config := bigcache.DefaultConfig(8 * time.Hour)
	config.CleanWindow = 1 * time.Minute
	config.HardMaxCacheSize = 1024

	cache2, err := bigcache.New(context.Background(), config)
	if err != nil {
		log.Fatalln(err)
	}
	a.UserIDToTokenCache = cache2
}
*/

func (a *TeslaAPIImpl) GetTokens(userID string, code string, redirectURI string) (*TeslaAPITokenReponse, error) {
	target := "https://auth.tesla.com/oauth2/v3/token"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", GetConfig().TeslaClientID)
	data.Set("client_secret", GetConfig().TeslaClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("audience", GetConfig().TeslaAudience)
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

	// Cache token
	//a.UserIDToTokenCache.Set(userID, []byte(m.AccessToken))

	return &m, nil
}

func (a *TeslaAPIImpl) RefreshToken(userID string, refreshToken string) (*TeslaAPITokenReponse, error) {
	target := "https://auth.tesla.com/oauth2/v3/token"
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", GetConfig().TeslaClientID)
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

	// Cache token
	//a.UserIDToTokenCache.Set(userID, []byte(m.AccessToken))

	return &m, nil
}

/*
func (a *TeslaAPIImpl) GetOrRefreshAccessToken(userID string) string {
	//log.Printf("GetOrRefreshAccessToken() with userID %s\n", userID)
	//debug.PrintStack()
	accessToken := a.GetCachedAccessToken(userID)
	if accessToken == "" {
		user := GetDB().GetUser(userID)
		if user == nil {
			log.Printf("user not found: %s\n", userID)
			return ""
		}
		token, err := a.RefreshToken(userID, user.TeslaRefreshToken)
		if err != nil {
			log.Println(err)
			return ""
		}
		user.TeslaRefreshToken = token.RefreshToken
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
*/

func (a *TeslaAPIImpl) InitSession(accessToken string, vin string) (*vehicle.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	acct, err := account.New(accessToken, "chargebot.io/0.0.1")
	if err != nil {
		return nil, err
	}
	car, err := acct.GetVehicle(ctx, vin, GetConfig().TeslaPrivateKey, nil)
	if err != nil {
		return nil, err
	}
	if err := car.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to vehicle: %s", err.Error())
	}
	if err := car.StartSession(ctx, []universalmessage.Domain{universalmessage.Domain_DOMAIN_INFOTAINMENT}); err != nil {
		return nil, fmt.Errorf("failed to perform handshake with vehicle: %s", err.Error())
	}
	return car, nil
}

func (a *TeslaAPIImpl) ListVehicles(accessToken string) ([]TeslaAPIVehicleEntity, error) {
	r, _ := http.NewRequest("GET", _configInstance.TeslaAudience+"/api/1/vehicles", nil)

	resp, err := RetryHTTPJSONRequest(r, accessToken)
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

func (a *TeslaAPIImpl) ChargeStart(car *vehicle.Vehicle) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := car.ChargeStart(ctx)
	if err != nil && (strings.Contains(err.Error(), "already_started") || strings.Contains(err.Error(), "is_charging")) {
		return nil
	}
	return err
}

func (a *TeslaAPIImpl) ChargeStop(car *vehicle.Vehicle) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := car.ChargeStop(ctx)
	if err != nil && strings.Contains(err.Error(), "not_charging") {
		return nil
	}
	return err
}

func (a *TeslaAPIImpl) SetChargeLimit(car *vehicle.Vehicle, limitPercent int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := car.ChangeChargeLimit(ctx, int32(limitPercent))
	if err != nil && strings.Contains(err.Error(), "already_set") {
		return nil
	}
	return err
}

func (a *TeslaAPIImpl) SetChargeAmps(car *vehicle.Vehicle, amps int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := car.SetChargingAmps(ctx, int32(amps))
	if err != nil && strings.Contains(err.Error(), "already_set") {
		return nil
	}
	return err
}

func (a *TeslaAPIImpl) GetVehicleData(accessToken string, vin string) (*TeslaAPIVehicleData, error) {
	target := GetConfig().TeslaAudience + "/api/1/vehicles/" + vin + "/vehicle_data"
	r, _ := http.NewRequest("GET", target, nil)

	resp, err := RetryHTTPJSONRequest(r, accessToken)
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

func (a *TeslaAPIImpl) Wakeup(accessToken string, vin string) error {
	target := GetConfig().TeslaAudience + "/api/1/vehicles/" + vin + "/wake_up"
	r, _ := http.NewRequest("POST", target, nil)

	_, err := RetryHTTPJSONRequest(r, accessToken)
	if err != nil {
		// TODO
		log.Println(err)
		return err
	}

	// wait a few seconds to assure vehicle is online
	time.Sleep(20 * time.Second)

	return nil
}

func (a *TeslaAPIImpl) CreateTelemetryConfig(accessToken string, vin string) error {
	config := TeslaAPITelemetryConfigCreate{
		VINs: []string{vin},
		Config: TeslaAPITelemetryConfig{
			Hostname:   GetConfig().TeslaTelemetryHost,
			Port:       443,
			CA:         GetConfig().TeslaTelemetryCA,
			Expiration: time.Now().UTC().AddDate(0, 10, 0).Unix(),
			Fields: map[string]TeslaAPITelemetryField{
				"ChargeState":     {IntervalSeconds: 60},
				"Soc":             {IntervalSeconds: 60},
				"Location":        {IntervalSeconds: 60},
				"ChargeLimitSoc":  {IntervalSeconds: 60},
				"ChargeAmps":      {IntervalSeconds: 60},
				"ChargePortLatch": {IntervalSeconds: 60},
			},
			AlertTypes: []string{"service"},
		},
	}
	json, err := json.Marshal(config)
	if err != nil {
		return err
	}

	target := GetConfig().TeslaAudience + "/api/1/vehicles/fleet_telemetry_config"
	r, _ := http.NewRequest("POST", target, bytes.NewReader(json))

	resp, err := RetryHTTPJSONRequest(r, accessToken)
	if err != nil {
		log.Println(err)
		return err
	}

	s, _ := DebugGetResponseBody(resp.Body)
	log.Println(s)

	return nil
}

func (a *TeslaAPIImpl) DeleteTelemetryConfig(accessToken string, vin string) error {
	target := GetConfig().TeslaAudience + "/api/1/vehicles/" + vin + "/fleet_telemetry_config"
	r, _ := http.NewRequest("DELETE", target, nil)

	_, err := RetryHTTPJSONRequest(r, accessToken)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (a *TeslaAPIImpl) GetTelemetryConfig(accessToken string, vin string) error {
	target := GetConfig().TeslaAudience + "/api/1/vehicles/" + vin + "/fleet_telemetry_config"
	r, _ := http.NewRequest("GET", target, nil)

	resp, err := RetryHTTPJSONRequest(r, accessToken)
	if err != nil {
		log.Println(err)
		return err
	}

	s, _ := DebugGetResponseBody(resp.Body)
	log.Println(s)

	return nil
}
