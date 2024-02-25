package main

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeslaAuthRouter_initThirdParty_noBearer(t *testing.T) {
	t.Cleanup(ResetTestDB)

	req := newHTTPRequest("GET", "/api/1/auth/tesla/init3rdparty", "", nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestTeslaAuthRouter_initThirdParty(t *testing.T) {
	t.Cleanup(ResetTestDB)

	bearer := getTestJWT("abc")
	req := newHTTPRequest("GET", "/api/1/auth/tesla/init3rdparty", bearer, nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusOK, res.Code)

	var m TeslaAuthRouterInitRequest
	err := UnmarshalBody(res.Result().Body, &m)
	assert.Nil(t, err)
	assert.Contains(t, m.URL, "https://auth.tesla.com/oauth2/v3/authorize")
}

func TestTeslaAuthRouter_callback_noBearer(t *testing.T) {
	t.Cleanup(ResetTestDB)

	req := newHTTPRequest("GET", "/api/1/auth/tesla/callback?state=foobar", "", nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestTeslaAuthRouter_callback_invalidState(t *testing.T) {
	t.Cleanup(ResetTestDB)

	bearer := getTestJWT("abc")
	req := newHTTPRequest("GET", "/api/1/auth/tesla/callback?state=foobar", bearer, nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusNotFound, res.Code)
}

func TestTeslaAuthRouter_callback_invalidCode(t *testing.T) {
	t.Cleanup(ResetTestDB)
	TeslaAPIInstance = &TeslaAPIImpl{}
	TeslaAPIInstance.InitTokenCache()

	bearer := getTestJWT("abc")
	req := newHTTPRequest("GET", "/api/1/auth/tesla/init3rdparty", bearer, nil)
	res := executeTestRequest(req)

	var m TeslaAuthRouterInitRequest
	err := UnmarshalBody(res.Result().Body, &m)
	assert.Nil(t, err)

	redirectURI, err := url.Parse(m.URL)
	assert.Nil(t, err)
	redirectParams, _ := url.ParseQuery(redirectURI.RawQuery)
	state := redirectParams.Get("state")
	assert.NotEmpty(t, state)

	req = newHTTPRequest("GET", "/api/1/auth/tesla/callback?state="+state, bearer, nil)
	res = executeTestRequest(req)
	assert.Equal(t, http.StatusBadRequest, res.Code)
}
