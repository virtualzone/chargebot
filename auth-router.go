package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

type AuthRouter struct {
}

type LoginResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	User         User   `json:"user"`
}

func (router *AuthRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/login", router.login).Methods("GET")
	s.HandleFunc("/callback", router.callback).Methods("GET")
	s.HandleFunc("/refresh", router.refresh).Methods("POST")
	s.HandleFunc("/tokenvalid", router.isTokenValid).Methods("GET")
	s.HandleFunc("/me", router.getMe).Methods("GET")
}

func (router *AuthRouter) login(w http.ResponseWriter, r *http.Request) {
	state := GetDB().CreateAuthCode()
	http.Redirect(w, r, GetOIDCProvider().OAuthConfig.AuthCodeURL(state), http.StatusFound)
}

func (router *AuthRouter) callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if !GetDB().IsValidAuthCode(state) {
		SendNotFound(w)
		return
	}
	code := r.URL.Query().Get("code")
	oauth2Token, err := GetOIDCProvider().OAuthConfig.Exchange(GetOIDCProvider().Context, code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	idToken, _, err := GetOIDCProvider().VerifyAuthHeader(oauth2Token.AccessToken)
	if err != nil {
		http.Error(w, "Failed to verify auth header: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims: "+err.Error(), http.StatusInternalServerError)
		return
	}
	user := GetDB().GetUser(idToken.Subject)
	if user == nil {
		user = &User{
			ID: idToken.Subject,
		}
		GetDB().CreateUpdateUser(user)
	}

	token := GetDB().GetAPIToken(user.ID)
	if token == "" {
		password := GeneratePassword(16, true, true)
		GetDB().CreateAPIToken(user.ID, password)
	}

	loginResponse := LoginResponse{
		AccessToken:  oauth2Token.AccessToken,
		RefreshToken: oauth2Token.RefreshToken,
		User:         *user,
	}
	SendJSON(w, loginResponse)
}

func (router *AuthRouter) refresh(w http.ResponseWriter, r *http.Request) {
	var m TeslaAPITokenReponse
	if err := UnmarshalValidateBody(r.Body, &m); err != nil {
		SendBadRequest(w)
		return
	}
	t := &oauth2.Token{RefreshToken: m.RefreshToken}
	ts := GetOIDCProvider().OAuthConfig.TokenSource(GetOIDCProvider().Context, t)
	oauth2Token, err := ts.Token()
	if err != nil {
		http.Error(w, "Failed to refresh access token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	idToken, _, err := GetOIDCProvider().VerifyAuthHeader(oauth2Token.AccessToken)
	if err != nil {
		http.Error(w, "Failed to verify auth header: "+err.Error(), http.StatusInternalServerError)
		return
	}
	user := GetDB().GetUser(idToken.Subject)
	if user == nil {
		http.Error(w, "User not found: "+idToken.Subject, http.StatusInternalServerError)
		return
	}
	loginResponse := LoginResponse{
		AccessToken:  oauth2Token.AccessToken,
		RefreshToken: oauth2Token.RefreshToken,
		User:         *user,
	}
	SendJSON(w, loginResponse)
}

func (router *AuthRouter) isTokenValid(w http.ResponseWriter, r *http.Request) {
	authToken := GetAuthTokenFromRequest(r)
	if authToken == "" {
		SendJSON(w, false)
		return
	}
	_, _, err := GetOIDCProvider().VerifyAuthHeader(authToken)
	if err != nil {
		SendJSON(w, false)
		return
	}
	SendJSON(w, true)
}

func (router *AuthRouter) getMe(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		SendUnauthorized(w)
		return
	}
	user := GetDB().GetUser(userID)
	SendJSON(w, user)
}
