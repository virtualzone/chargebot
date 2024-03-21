package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
	token := a.GetOrRefreshAccessToken()
	payload := AccessTokenRequest{
		PasswordProtectedRequest: PasswordProtectedRequest{
			Password: "",
		},
		AccessToken: token,
	}
	json, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	target := GetConfig().CmdEndpoint + "/list_vehicles"
	r, _ := http.NewRequest("POST", target, bytes.NewReader(json))

	resp, err := RetryHTTPJSONRequest(r, a.GetOrRefreshAccessToken())
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var m []TeslaAPIVehicleEntity
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func (a *TeslaAPIProxy) ChargeStart(vin string) error {
	// TODO
	return nil
}

func (a *TeslaAPIProxy) ChargeStop(vin string) error {
	// TODO
	return nil
}

func (a *TeslaAPIProxy) SetChargeLimit(vin string, limitPercent int) error {
	// TODO
	return nil
}

func (a *TeslaAPIProxy) SetChargeAmps(vin string, amps int) error {
	// TODO
	return nil
}

func (a *TeslaAPIProxy) GetVehicleData(vin string) (*TeslaAPIVehicleData, error) {
	// TODO
	return nil, nil
}

func (a *TeslaAPIProxy) Wakeup(vin string) error {
	// TODO
	return nil
}

func (a *TeslaAPIProxy) CreateTelemetryConfig(vin string) error {
	// TODO
	return nil
}

func (a *TeslaAPIProxy) DeleteTelemetryConfig(vin string) error {
	// TODO
	return nil
}
