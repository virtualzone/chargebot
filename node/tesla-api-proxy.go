package main

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TeslaAPI struct {
	accessToken string
	expiry      int64
}

type TeslaAPIVehicleEntity struct {
	VehicleID   int    `json:"vehicle_id"`
	VIN         string `json:"vin"`
	DisplayName string `json:"display_name"`
}

type TeslaAPITokenReponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

func (a *TeslaAPI) RefreshToken(refreshToken string) (*TeslaAPITokenReponse, error) {
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

func (a *TeslaAPI) GetOrRefreshAccessToken() string {
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

func (a *TeslaAPI) GetCachedAccessToken() string {
	if a.accessToken == "" {
		return ""
	}
	limit := time.Now().UTC().Add(time.Minute * 5).Unix()
	if a.expiry <= limit {
		return ""
	}
	return a.accessToken
}

func (a *TeslaAPI) ListVehicles() ([]TeslaAPIVehicleEntity, error) {
	target := GetConfig().CmdEndpoint + "/list_vehicles"
	r, _ := http.NewRequest("POST", target, nil)

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
