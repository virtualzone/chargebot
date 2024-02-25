package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-playground/validator"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type contextKey string

var (
	contextKeyUserID     = contextKey("UserID")
	contextKeyAuthHeader = contextKey("AuthHeader")
	httpRouter           *mux.Router
)

type Route interface {
	SetupRoutes(s *mux.Router)
}

func SendNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}

func SendBadRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
}

func SendUnauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
}

func SendForbidden(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
}

func SendInternalServerError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
}

func SendJSON(w http.ResponseWriter, v interface{}) {
	json, err := json.Marshal(v)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func SendMethodNotAllowed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func UnmarshalBody(r io.ReadCloser, o interface{}) error {
	if r == nil {
		return errors.New("body is NIL")
	}
	defer r.Close()
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(body, &o); err != nil {
		return err
	}
	return nil
}

func DebugGetResponseBody(r io.ReadCloser) (string, error) {
	if r == nil {
		return "", errors.New("body is NIL")
	}
	defer r.Close()
	body, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func GetAuthTokenFromRequest(r *http.Request) string {
	authHeader := r.Context().Value(contextKeyAuthHeader)
	if authHeader == nil {
		return ""
	}
	return authHeader.(string)
}

func GetUserIDFromRequest(r *http.Request) string {
	userID := r.Context().Value(contextKeyUserID)
	if userID == nil {
		return ""
	}
	return userID.(string)
}

func UnmarshalValidateBody(r io.ReadCloser, o interface{}) error {
	err := UnmarshalBody(r, &o)
	if err != nil {
		return err
	}
	err = validator.New().Struct(o)
	if err != nil {
		return err
	}
	return nil
}

func VerifyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer := r.Header.Get("Authorization")
		authHeader := ""
		userID := ""
		tokenValid := false
		if strings.Index(bearer, "Bearer ") == 0 {
			h := strings.TrimPrefix(bearer, "Bearer ")
			if h != "" {
				if OIDCTestingMode {
					token, err := jwt.Parse(h, func(token *jwt.Token) (interface{}, error) {
						return []byte(OIDCTestingSecret), nil
					})
					log.Println(err)
					if err == nil {
						claims, _ := token.Claims.(jwt.MapClaims)
						resClaims := Claims{}
						for k, v := range claims {
							resClaims[k] = v
						}
						idToken := oidc.IDToken{
							Subject: resClaims["sub"].(string),
						}
						tokenValid = true
						authHeader = h
						userID = idToken.Subject
					}
				} else {
					idToken, _, err := GetOIDCProvider().VerifyAuthHeader(h)
					if err == nil {
						tokenValid = true
						authHeader = h
						userID = idToken.Subject
					}
				}
			}
		}

		if !tokenValid {
			authURLs := []string{
				"/api/1/auth/tesla/",
				"/api/1/tesla/",
				"/api/1/ctrl/",
			}
			url := r.URL.RequestURI()
			for _, authURL := range authURLs {
				authURL = strings.TrimSpace(authURL)
				authURL = strings.TrimSuffix(authURL, "/")
				if authURL != "" && (url == authURL || strings.HasPrefix(url, authURL+"/")) {
					SendUnauthorized(w)
					return
				}
			}
		}

		ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
		ctx = context.WithValue(ctx, contextKeyAuthHeader, authHeader)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func InitHTTPRouter() {
	router := mux.NewRouter()
	routers := make(map[string]Route)
	routers["/api/1/auth/tesla/"] = &TeslaAuthRouter{}
	routers["/api/1/auth/"] = &AuthRouter{}
	routers["/api/1/tesla/"] = &TeslaRouter{}
	routers["/api/1/user/"] = &UserRouter{}
	routers["/api/1/ctrl/"] = &ManualControlRouter{}

	for prefix, route := range routers {
		subRouter := router.PathPrefix(prefix).Subrouter()
		route.SetupRoutes(subRouter)
	}

	if GetConfig().DevProxy {
		target, _ := url.Parse("http://localhost:3000")
		proxy := httputil.NewSingleHostReverseProxy(target)
		router.PathPrefix("/").Handler(proxy)
	} else {
		fs := http.FileServer(http.Dir("./static"))
		router.PathPrefix("/").Handler(fs)
	}

	router.Use(VerifyAuthMiddleware)

	httpRouter = router
}

func ServeHTTP() {

	log.Println("Initializing REST services...")
	httpServer := &http.Server{
		Addr:         "0.0.0.0:8080",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      httpRouter,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatal(err)
			os.Exit(-1)
		}
	}()
	log.Println("HTTP Server listening")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	httpServer.Shutdown(ctx)
}
