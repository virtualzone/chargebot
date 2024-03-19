package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeslaRouter_listVehicles_noBearer(t *testing.T) {
	t.Cleanup(ResetTestDB)

	req := newHTTPRequest("GET", "/api/1/tesla/vehicles", "", nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestTeslaRouter_listVehicles_unknownBearer(t *testing.T) {
	t.Cleanup(ResetTestDB)
	TeslaAPIInstance = &TeslaAPIImpl{}
	//TeslaAPIInstance.InitTokenCache()

	bearer := getTestJWT("12345")
	req := newHTTPRequest("GET", "/api/1/tesla/vehicles", bearer, nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusUnauthorized, res.Code)
}
