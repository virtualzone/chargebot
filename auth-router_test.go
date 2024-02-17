package main

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthRouter_initThirdParty(t *testing.T) {
	t.Cleanup(ResetTestDB)

	req := newHTTPRequest("GET", "/api/1/auth/init3rdparty", "", nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusTemporaryRedirect, res.Code)
	assert.Contains(t, res.Header().Get("Location"), "https://auth.tesla.com/oauth2/v3/authorize")
}

func TestAuthRouter_callback_invalidState(t *testing.T) {
	t.Cleanup(ResetTestDB)

	req := newHTTPRequest("GET", "/api/1/auth/callback?state=foobar", "", nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusNotFound, res.Code)
}

func TestAuthRouter_callback_invalidCode(t *testing.T) {
	t.Cleanup(ResetTestDB)
	TeslaAPIInstance = &TeslaAPIImpl{}
	TeslaAPIInstance.InitTokenCache()

	req := newHTTPRequest("GET", "/api/1/auth/init3rdparty", "", nil)
	res := executeTestRequest(req)
	redirectURI, err := url.Parse(res.Header().Get("Location"))
	assert.Nil(t, err)
	redirectParams, _ := url.ParseQuery(redirectURI.RawQuery)
	state := redirectParams.Get("state")
	assert.NotEmpty(t, state)

	req = newHTTPRequest("GET", "/api/1/auth/callback?state="+state, "", nil)
	res = executeTestRequest(req)
	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func TestAuthRouter_isTokenValid_noBearer(t *testing.T) {
	t.Cleanup(ResetTestDB)

	req := newHTTPRequest("GET", "/api/1/auth/tokenvalid", "", nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusOK, res.Code)
	var boolResult bool
	err := UnmarshalBody(res.Result().Body, boolResult)
	assert.Nil(t, err)
	assert.False(t, boolResult)
}

func TestAuthRouter_isTokenValid_valid(t *testing.T) {
	t.Cleanup(ResetTestDB)
	bearer := getTestJWT("12345")
	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("IsKnownAccessToken", bearer).Return(true)

	req := newHTTPRequest("GET", "/api/1/auth/tokenvalid", bearer, nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusOK, res.Code)
	var boolResult bool
	err := UnmarshalBody(res.Result().Body, &boolResult)
	assert.Nil(t, err)
	assert.True(t, boolResult)
}
func TestAuthRouter_isTokenValid_expired(t *testing.T) {
	t.Cleanup(ResetTestDB)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().UTC().AddDate(0, 0, -1).Unix(),
		"iat": time.Now().UTC().Unix(),
		"sub": "12345",
	})
	bearer, _ := token.SignedString([]byte("sample-secret"))
	api, _ := TeslaAPIInstance.(*TeslaAPIMock)
	api.On("IsKnownAccessToken", bearer).Return(true)

	req := newHTTPRequest("GET", "/api/1/auth/tokenvalid", bearer, nil)
	res := executeTestRequest(req)

	assert.Equal(t, http.StatusOK, res.Code)
	var boolResult bool
	err := UnmarshalBody(res.Result().Body, &boolResult)
	assert.Nil(t, err)
	assert.False(t, boolResult)
}
