package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
)

type TokenReponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

type AuthRouter struct {
}

func (router *AuthRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/init3rdparty", router.initThirdParty).Methods("GET")
	s.HandleFunc("/callback", router.callback).Methods("GET")
	s.HandleFunc("/refresh", router.refresh).Methods("POST")
}

func (router *AuthRouter) getRedirectURI() string {
	if strings.Index(GetConfig().Hostname, "localhost") != -1 {
		return "http://" + GetConfig().Hostname + "/callback"
	}
	return "https://" + GetConfig().Hostname + "/callback"
}

func (router *AuthRouter) initThirdParty(w http.ResponseWriter, r *http.Request) {
	code := CreateAuthCode()

	scope := []string{
		"openid",
		"vehicle_device_data",
		"vehicle_cmds",
		"vehicle_charging_cmds",
		"offline_access",
	}
	params := url.Values{}
	params.Add("client_id", GetConfig().ClientID)
	params.Add("prompt", "login")
	params.Add("redirect_uri", router.getRedirectURI())
	params.Add("response_type", "code")
	params.Add("scope", strings.Join(scope, " "))
	params.Add("state", code)

	url := "https://auth.tesla.com/oauth2/v3/authorize?" + params.Encode()
	w.Header().Add("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (router *AuthRouter) callback(w http.ResponseWriter, r *http.Request) {
	SendJSON(w, TokenReponse{AccessToken: "abc", RefreshToken: "def"})
	return

	state := r.URL.Query().Get("state")
	if !IsValidAuthCode(state) {
		SendNotFound(w)
		return
	}
	tokens, err := router.getTokens(r.URL.Query().Get("code"))
	if err != nil {
		log.Println(err)
		SendBadRequest(w)
		return
	}
	// TODO Save somehow?!
	SendJSON(w, tokens)
}

func (router *AuthRouter) refresh(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func (router *AuthRouter) getTokens(code string) (*TokenReponse, error) {
	target := "https://auth.tesla.com/oauth2/v3/token"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", GetConfig().ClientID)
	data.Set("client_secret", GetConfig().ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", router.getRedirectURI())
	data.Set("audience", GetConfig().Audience)
	r, _ := http.NewRequest("POST", target, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		// TODO
		log.Println(err)
		return nil, err
	}

	var m TokenReponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
