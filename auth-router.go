package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"
)

func GetAuthRouterInitThirdParty(w http.ResponseWriter, r *http.Request) {
	code := CreateAuthCode()

	redirectURI := "https://" + GetConfig().Hostname + "/api/1/auth/callback"
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
	params.Add("redirect_uri", redirectURI)
	params.Add("response_type", "code")
	params.Add("scope", strings.Join(scope, " "))
	params.Add("state", code)

	url := "https://auth.tesla.com/oauth2/v3/authorize?" + params.Encode()
	w.Header().Add("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func GetAuthRouterCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if !IsValidAuthCode(code) {
		SendNotFound(w)
		return
	}
	getTokens(code)
}

func getTokens(code string) {
	target := "https://auth.tesla.com/oauth2/v3/token"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", GetConfig().ClientID)
	data.Set("client_secret", "")
	data.Set("code", code)
	data.Set("redirect_uri", "https://"+GetConfig().Hostname+"/api/1/auth/callback")
	data.Set("audience", GetConfig().Audience)
	r, _ := http.NewRequest("POST", target, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, _ := client.Do(r)
	log.Println(resp)
}
