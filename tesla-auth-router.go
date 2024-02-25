package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type TeslaAuthRouter struct {
}

type TeslaAuthRouterInitRequest struct {
	URL string `json:"url"`
}

func (router *TeslaAuthRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/init3rdparty", router.initThirdParty).Methods("GET")
	s.HandleFunc("/callback", router.callback).Methods("GET")
}

func (router *TeslaAuthRouter) getRedirectURI() string {
	if strings.Contains(GetConfig().Hostname, "localhost") {
		return "http://" + GetConfig().Hostname + "/tesla-callback"
	}
	return "https://" + GetConfig().Hostname + "/tesla-callback"
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

	res := TeslaAuthRouterInitRequest{
		URL: "https://auth.tesla.com/oauth2/v3/authorize?" + params.Encode(),
	}
	SendJSON(w, res)
}

func (router *TeslaAuthRouter) callback(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	state := r.URL.Query().Get("state")
	if !GetDB().IsValidAuthCode(state) {
		SendNotFound(w)
		return
	}
	tokens, err := GetTeslaAPI().GetTokens(userID, r.URL.Query().Get("code"), router.getRedirectURI())
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

	user := GetDB().GetUser(userID)
	if user == nil {
		SendNotFound(w)
		return
	}

	user.TeslaRefreshToken = tokens.RefreshToken
	user.TeslaUserID = sub
	GetDB().CreateUpdateUser(user)

	loginResponse := LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         *user,
	}
	SendJSON(w, loginResponse)
}
