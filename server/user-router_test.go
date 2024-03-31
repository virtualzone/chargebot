package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	. "github.com/virtualzone/chargebot/goshared"
)

func TestUserRouter_ping_noPass(t *testing.T) {
	t.Cleanup(ResetTestDB)

	token := uuid.NewString()

	req := newHTTPRequest("POST", "/api/1/user/"+token+"/ping", "", nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func TestUserRouter_ping_unauthorized(t *testing.T) {
	t.Cleanup(ResetTestDB)

	token := uuid.NewString()
	payload := `{"password": "1234"}`

	req := newHTTPRequest("POST", "/api/1/user/"+token+"/ping", "", strings.NewReader(payload))
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestUserRouter_ping_ok(t *testing.T) {
	t.Cleanup(ResetTestDB)

	user := &User{
		ID:            uuid.NewString(),
		HomeLatitude:  0.0,
		HomeLongitude: 0.0,
		HomeRadius:    100,
		Region:        RegionCodeEU,
	}
	GetDB().CreateUpdateUser(user)
	password := GeneratePassword(16, true, false)
	token := GetDB().CreateAPIToken(user.ID, password)

	payload := `{"password": "` + password + `"}`
	req := newHTTPRequest("POST", "/api/1/user/"+token+"/ping", "", strings.NewReader(payload))
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusOK, res.Code)
}

func TestUserRouter_listVehicles_checkRegionEU(t *testing.T) {
	t.Cleanup(ResetTestDB)

	user := &User{
		ID:            uuid.NewString(),
		HomeLatitude:  0.0,
		HomeLongitude: 0.0,
		HomeRadius:    100,
		Region:        RegionCodeEU,
	}
	GetDB().CreateUpdateUser(user)
	password := GeneratePassword(16, true, false)
	token := GetDB().CreateAPIToken(user.ID, password)

	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("ListVehicles", "https://fleet-api.prd.eu.vn.cloud.tesla.com", "abc123").Return([]TeslaAPIVehicleEntity{}, nil)

	payload := `{"password": "` + password + `", "access_token": "abc123"}`
	req := newHTTPRequest("POST", "/api/1/user/"+token+"/list_vehicles", "", strings.NewReader(payload))
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusOK, res.Code)
}

func TestUserRouter_listVehicles_checkRegionNA(t *testing.T) {
	t.Cleanup(ResetTestDB)

	user := &User{
		ID:            uuid.NewString(),
		HomeLatitude:  0.0,
		HomeLongitude: 0.0,
		HomeRadius:    100,
		Region:        RegionCodeNA,
	}
	GetDB().CreateUpdateUser(user)
	password := GeneratePassword(16, true, false)
	token := GetDB().CreateAPIToken(user.ID, password)

	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("ListVehicles", "https://fleet-api.prd.na.vn.cloud.tesla.com", "abc123").Return([]TeslaAPIVehicleEntity{}, nil)

	payload := `{"password": "` + password + `", "access_token": "abc123"}`
	req := newHTTPRequest("POST", "/api/1/user/"+token+"/list_vehicles", "", strings.NewReader(payload))
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusOK, res.Code)
}
