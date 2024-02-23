package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type TeslaAuthRouter struct {
}

type LoginResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	UserID       string `json:"user_id"`
}

func (router *TeslaAuthRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/init3rdparty", router.initThirdParty).Methods("GET")
	s.HandleFunc("/callback", router.callback).Methods("GET")
	s.HandleFunc("/refresh", router.refresh).Methods("POST")
	s.HandleFunc("/tokenvalid", router.isTokenValid).Methods("GET")
}

func (router *TeslaAuthRouter) getRedirectURI() string {
	if strings.Contains(GetConfig().Hostname, "localhost") {
		return "http://" + GetConfig().Hostname + "/callback"
	}
	return "https://" + GetConfig().Hostname + "/callback"
}

func (router *TeslaAuthRouter) initThirdParty(w http.ResponseWriter, r *http.Request) {
	code := GetDB().CreateAuthCode()

	scope := []string{
		"openid",
		"vehicle_device_data",
		"vehicle_cmds",
		"vehicle_charging_cmds",
		"offline_access",
	}
	params := url.Values{}
	params.Add("client_id", GetConfig().TeslaClientID)
	params.Add("prompt", "login")
	params.Add("redirect_uri", router.getRedirectURI())
	params.Add("response_type", "code")
	params.Add("scope", strings.Join(scope, " "))
	params.Add("state", code)

	url := "https://auth.tesla.com/oauth2/v3/authorize?" + params.Encode()
	w.Header().Add("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (router *TeslaAuthRouter) callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if !GetDB().IsValidAuthCode(state) {
		SendNotFound(w)
		return
	}
	tokens, err := GetTeslaAPI().GetTokens(r.URL.Query().Get("code"), router.getRedirectURI())
	if err != nil {
		log.Println(err)
		SendBadRequest(w)
		return
	}

	parsedToken, _ := jwt.Parse(tokens.AccessToken, nil)
	if parsedToken == nil || parsedToken.Claims == nil {
		SendInternalServerError(w)
		return
	}
	sub, _ := parsedToken.Claims.GetSubject()
	user := &User{
		ID:           sub,
		RefreshToken: tokens.RefreshToken,
	}
	GetDB().CreateUpdateUser(user)

	loginResponse := LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		UserID:       user.ID,
	}
	SendJSON(w, loginResponse)
}

func (router *TeslaAuthRouter) refresh(w http.ResponseWriter, r *http.Request) {
	var m TeslaAPITokenReponse
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}

	tokens, err := GetTeslaAPI().RefreshToken(m.RefreshToken)
	if err != nil {
		log.Println(err)
		SendBadRequest(w)
		return
	}

	parsedToken, _ := jwt.Parse(tokens.AccessToken, nil)
	if parsedToken == nil || parsedToken.Claims == nil {
		SendInternalServerError(w)
		return
	}
	sub, _ := parsedToken.Claims.GetSubject()
	user := &User{
		ID:           sub,
		RefreshToken: tokens.RefreshToken,
	}
	GetDB().CreateUpdateUser(user)

	loginResponse := LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		UserID:       "",
	}
	SendJSON(w, loginResponse)
}

func (router *TeslaAuthRouter) isTokenValid(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	if authToken == "" {
		SendJSON(w, false)
		return
	}

	parsedToken, _ := jwt.Parse(authToken, nil)
	if parsedToken == nil || parsedToken.Claims == nil {
		SendInternalServerError(w)
		return
	}

	exp, err := parsedToken.Claims.GetExpirationTime()
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	now := time.Now().UTC()
	if exp.Before(now) {
		SendJSON(w, false)
		return
	}
	SendJSON(w, true)
}