package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/teslamotors/vehicle-command/pkg/account"
	"github.com/teslamotors/vehicle-command/pkg/protocol/protobuf/universalmessage"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
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
	InitSession(authToken string, vehicle *Vehicle, wakeUp bool) (*vehicle.Vehicle, error)
	ListVehicles(authToken string) ([]TeslaAPIVehicleEntity, error)
	ChargeStart(car *vehicle.Vehicle) error
	ChargeStop(car *vehicle.Vehicle) error
	SetChargeLimit(car *vehicle.Vehicle, limitPercent int) error
	SetChargeAmps(car *vehicle.Vehicle, amps int) error
	GetVehicleData(authToken string, vehicle *Vehicle) (*TeslaAPIVehicleData, error)
	Wakeup(authToken string, vehicle *Vehicle) error
}

type TeslaAPIImpl struct {
	TokenCache         *bigcache.BigCache
	UserIDToTokenCache *bigcache.BigCache
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
}

func (a *TeslaAPIImpl) IsKnownAccessToken(token string) bool {
	v, err := a.TokenCache.Get(token)
	return err == nil && v != nil
}

func (a *TeslaAPIImpl) GetTokens(code string, redirectURI string) (*TeslaAPITokenReponse, error) {
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

func (a *TeslaAPIImpl) InitSession(authToken string, vehicle *Vehicle, wakeUp bool) (*vehicle.Vehicle, error) {
	if wakeUp {
		a.Wakeup(authToken, vehicle)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	acct, err := account.New(authToken, "chargebot.io/0.0.1")
	if err != nil {
		return nil, err
	}
	car, err := acct.GetVehicle(ctx, vehicle.VIN, GetConfig().TeslaPrivateKey, nil)
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

func (a *TeslaAPIImpl) ListVehicles(authToken string) ([]TeslaAPIVehicleEntity, error) {
	r, _ := http.NewRequest("GET", _configInstance.TeslaAudience+"/api/1/vehicles", nil)

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

func (a *TeslaAPIImpl) ChargeStart(car *vehicle.Vehicle) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := car.ChargeStart(ctx)
	if err != nil && strings.Contains(err.Error(), "already_started") {
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

func (a *TeslaAPIImpl) GetVehicleData(authToken string, vehicle *Vehicle) (*TeslaAPIVehicleData, error) {
	target := GetConfig().TeslaAudience + "/api/1/vehicles/" + vehicle.VIN + "/vehicle_data"
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

func (a *TeslaAPIImpl) Wakeup(authToken string, vehicle *Vehicle) error {
	target := GetConfig().TeslaAudience + "/api/1/vehicles/" + vehicle.VIN + "/wake_up"
	r, _ := http.NewRequest("POST", target, nil)

	_, err := RetryHTTPJSONRequest(r, authToken)
	if err != nil {
		// TODO
		log.Println(err)
		return err
	}

	// wait a few seconds to assure vehicle is online
	time.Sleep(20 * time.Second)

	return nil
}
