package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	. "github.com/virtualzone/chargebot/goshared"
)

type TeslaAPI interface {
	RefreshToken(refreshToken string) (*TeslaAPITokenReponse, error)
	ListVehicles() ([]TeslaAPIVehicleEntity, error)
	ChargeStart(vin string) error
	ChargeStop(vin string) error
	SetChargeLimit(vin string, limitPercent int) error
	SetChargeAmps(vin string, amps int) error
	GetVehicleData(vin string) (*TeslaAPIVehicleData, error)
	Wakeup(vin string) error
	CreateTelemetryConfig(vin string) error
	DeleteTelemetryConfig(vin string) error
	RegisterVehicle(vin string) error
	UnregisterVehicle(vin string) error
}

type TeslaAPIProxy struct {
	accessToken string
	expiry      int64
}

func (a *TeslaAPIProxy) RefreshToken(refreshToken string) (*TeslaAPITokenReponse, error) {
	target := "https://auth.tesla.com/oauth2/v3/token"
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", GetConfig().TeslaClientID)
	data.Set("refresh_token", refreshToken)
	r, _ := http.NewRequest("POST", target, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := RetryHTTPRequest(r)
	if err != nil {
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
	a.accessToken = m.AccessToken
	exp, err := parsedToken.Claims.GetExpirationTime()
	if err == nil {
		a.expiry = exp.UTC().Unix()
	}
	GetDB().SetSetting(SettingRefreshToken, m.RefreshToken)

	return &m, nil
}

func (a *TeslaAPIProxy) GetOrRefreshAccessToken() string {
	accessToken := a.GetCachedAccessToken()
	if accessToken == "" {
		refreshToken := GetDB().GetSetting(SettingRefreshToken)
		token, err := a.RefreshToken(refreshToken)
		if err != nil {
			log.Println(err)
			return ""
		}
		accessToken = token.AccessToken
	}
	return accessToken
}

func (a *TeslaAPIProxy) GetCachedAccessToken() string {
	if a.accessToken == "" {
		return ""
	}
	limit := time.Now().UTC().Add(time.Minute * 5).Unix()
	if a.expiry <= limit {
		return ""
	}
	return a.accessToken
}

func (a *TeslaAPIProxy) ListVehicles() ([]TeslaAPIVehicleEntity, error) {
	payload := AccessTokenRequest{
		PasswordProtectedRequest: PasswordProtectedRequest{
			Password: GetConfig().TokenPassword,
		},
		AccessToken: a.GetOrRefreshAccessToken(),
	}

	resp, err := a.sendRequest("list_vehicles", payload)
	if err != nil {
		return nil, err
	}

	var m []TeslaAPIVehicleEntity
	if err := UnmarshalBody(resp.Body, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func (a *TeslaAPIProxy) ChargeStart(vin string) error {
	payload := AccessTokenRequest{
		PasswordProtectedRequest: PasswordProtectedRequest{
			Password: GetConfig().TokenPassword,
		},
		AccessToken: a.GetOrRefreshAccessToken(),
	}

	_, err := a.sendRequest(vin+"/charge_start", payload)
	if err != nil {
		return err
	}

	return nil
}

func (a *TeslaAPIProxy) ChargeStop(vin string) error {
	payload := AccessTokenRequest{
		PasswordProtectedRequest: PasswordProtectedRequest{
			Password: GetConfig().TokenPassword,
		},
		AccessToken: a.GetOrRefreshAccessToken(),
	}

	_, err := a.sendRequest(vin+"/charge_stop", payload)
	if err != nil {
		return err
	}

	return nil
}

func (a *TeslaAPIProxy) SetChargeLimit(vin string, limitPercent int) error {
	payload := SetChargeLimitRequest{
		AccessTokenRequest: AccessTokenRequest{
			PasswordProtectedRequest: PasswordProtectedRequest{
				Password: GetConfig().TokenPassword,
			},
			AccessToken: a.GetOrRefreshAccessToken(),
		},
		ChargeLimit: limitPercent,
	}

	_, err := a.sendRequest(vin+"/set_charge_limit", payload)
	if err != nil {
		return err
	}

	return nil
}

func (a *TeslaAPIProxy) SetChargeAmps(vin string, amps int) error {
	payload := SetChargeAmpsRequest{
		AccessTokenRequest: AccessTokenRequest{
			PasswordProtectedRequest: PasswordProtectedRequest{
				Password: GetConfig().TokenPassword,
			},
			AccessToken: a.GetOrRefreshAccessToken(),
		},
		Amps: amps,
	}

	_, err := a.sendRequest(vin+"/set_charge_amps", payload)
	if err != nil {
		return err
	}

	return nil
}

func (a *TeslaAPIProxy) GetVehicleData(vin string) (*TeslaAPIVehicleData, error) {
	payload := AccessTokenRequest{
		PasswordProtectedRequest: PasswordProtectedRequest{
			Password: GetConfig().TokenPassword,
		},
		AccessToken: a.GetOrRefreshAccessToken(),
	}

	resp, err := a.sendRequest(vin+"/vehicle_data", payload)
	if err != nil {
		return nil, err
	}

	var m TeslaAPIVehicleData
	if err := UnmarshalBody(resp.Body, &m); err != nil {
		return nil, err
	}

	return &m, nil
}

func (a *TeslaAPIProxy) Wakeup(vin string) error {
	payload := AccessTokenRequest{
		PasswordProtectedRequest: PasswordProtectedRequest{
			Password: GetConfig().TokenPassword,
		},
		AccessToken: a.GetOrRefreshAccessToken(),
	}

	_, err := a.sendRequest(vin+"/wakeup", payload)
	if err != nil {
		return err
	}

	return nil
}

func (a *TeslaAPIProxy) CreateTelemetryConfig(vin string) error {
	payload := AccessTokenRequest{
		PasswordProtectedRequest: PasswordProtectedRequest{
			Password: GetConfig().TokenPassword,
		},
		AccessToken: a.GetOrRefreshAccessToken(),
	}

	_, err := a.sendRequest(vin+"/create_telemetry_config", payload)
	if err != nil {
		return err
	}

	return nil
}

func (a *TeslaAPIProxy) DeleteTelemetryConfig(vin string) error {
	payload := AccessTokenRequest{
		PasswordProtectedRequest: PasswordProtectedRequest{
			Password: GetConfig().TokenPassword,
		},
		AccessToken: a.GetOrRefreshAccessToken(),
	}

	_, err := a.sendRequest(vin+"/delete_telemetry_config", payload)
	if err != nil {
		return err
	}

	return nil
}

func (a *TeslaAPIProxy) RegisterVehicle(vin string) error {
	payload := AccessTokenRequest{
		PasswordProtectedRequest: PasswordProtectedRequest{
			Password: GetConfig().TokenPassword,
		},
		AccessToken: a.GetOrRefreshAccessToken(),
	}

	_, err := a.sendRequest("vehicle_add/"+vin, payload)
	if err != nil {
		return err
	}

	return nil
}

func (a *TeslaAPIProxy) UnregisterVehicle(vin string) error {
	payload := AccessTokenRequest{
		PasswordProtectedRequest: PasswordProtectedRequest{
			Password: GetConfig().TokenPassword,
		},
		AccessToken: a.GetOrRefreshAccessToken(),
	}

	_, err := a.sendRequest("vehicle_delete/"+vin, payload)
	if err != nil {
		return err
	}

	return nil
}

func (a *TeslaAPIProxy) sendRequest(endpoint string, payload interface{}) (*http.Response, error) {
	json, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	target := GetConfig().CmdEndpoint + "/" + endpoint
	r, _ := http.NewRequest("POST", target, bytes.NewReader(json))

	resp, err := RetryHTTPJSONRequest(r, a.GetOrRefreshAccessToken())
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusInternalServerError {
			var m ErrorResponse
			if err := UnmarshalBody(resp.Body, &m); err == nil {
				return nil, fmt.Errorf("api error: %s", m.Error)
			}
		}
		return nil, fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	return resp, nil
}
