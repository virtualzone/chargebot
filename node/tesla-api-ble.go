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
	"github.com/teslamotors/vehicle-command/pkg/connector/ble"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
	. "github.com/virtualzone/chargebot/goshared"
)

type TeslaAPIBLE struct {
	accessToken string
	expiry      int64
}

func (a *TeslaAPIBLE) RefreshToken(refreshToken string) (*TeslaAPITokenReponse, error) {
	log.Println("Tesla API: Refreshing Access Token...")

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

func (a *TeslaAPIBLE) GetOrRefreshAccessToken() string {
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

func (a *TeslaAPIBLE) GetCachedAccessToken() string {
	if a.accessToken == "" {
		return ""
	}
	limit := time.Now().UTC().Add(time.Minute * 5).Unix()
	if a.expiry <= limit {
		return ""
	}
	return a.accessToken
}

func (a *TeslaAPIBLE) ListVehicles() ([]TeslaAPIVehicleEntity, error) {
	log.Println("Tesla API: List Vehicles...")

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

func (a *TeslaAPIBLE) ChargeStart(vin string) error {
	log.Println("Tesla API: Start Charge...")

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

func (a *TeslaAPIBLE) ChargeStop(vin string) error {
	log.Println("Tesla API: Stop Charge...")

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

func (a *TeslaAPIBLE) SetChargeLimit(vin string, limitPercent int) error {
	log.Printf("Tesla API: Set Charge Limit to % d ...\n", limitPercent)

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

func (a *TeslaAPIBLE) SetChargeAmps(vin string, amps int) error {
	log.Printf("Tesla API: Set Charge Amps to % d ...\n", amps)

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

func (a *TeslaAPIBLE) GetVehicleData(vin string) (*TeslaAPIVehicleData, error) {
	log.Println("Tesla API: Get Vehicle Data...")

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

func (a *TeslaAPIBLE) Wakeup(vin string) error {
	log.Println("Tesla BLE: Wake Up...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := ble.NewConnection(ctx, vin)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to connect to vehicle: %s", err))
	}
	defer conn.Close()

	car, err := vehicle.NewVehicle(conn, privateKey, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to connect to vehicle: %s", err))
	}

	if err := car.Connect(ctx); err != nil {
		return errors.New(fmt.Sprintf("failed to connect to vehicle: %s", err))
	}
	defer car.Disconnect()

	if err := car.StartSession(ctx, nil); err != nil {
		return errors.New(fmt.Sprintf("failed to perform handshake with vehicle: %s", err))
	}

	return nil
}

func (a *TeslaAPIBLE) CreateTelemetryConfig(vin string) error {
	log.Println("Tesla API: Create Telemetry Config...")

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

func (a *TeslaAPIBLE) DeleteTelemetryConfig(vin string) error {
	log.Println("Tesla API: Delete Telemetry Config...")

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

func (a *TeslaAPIBLE) RegisterVehicle(vin string) error {
	log.Println("Tesla API: Register Vehicle with chargebot.io...")

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

func (a *TeslaAPIBLE) UnregisterVehicle(vin string) error {
	log.Println("Tesla API: Unregister Vehicle with chargebot.io...")

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

func (a *TeslaAPIBLE) GetTelemetryState(vin string) (*PersistedTelemetryState, error) {
	payload := PasswordProtectedRequest{
		Password: GetConfig().TokenPassword,
	}

	resp, err := a.sendRequest(vin+"/state", payload)
	if err != nil {
		return nil, err
	}

	var m PersistedTelemetryState
	if err := UnmarshalBody(resp.Body, &m); err != nil {
		return nil, err
	}

	return &m, nil
}

func (a *TeslaAPIBLE) sendRequest(endpoint string, payload interface{}) (*http.Response, error) {
	json, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	target := GetConfig().CmdEndpoint + "/" + endpoint
	r, _ := http.NewRequest("POST", target, bytes.NewReader(json))

	resp, err := RetryHTTPJSONRequest(r, "")
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
