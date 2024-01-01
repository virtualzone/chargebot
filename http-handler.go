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

	"github.com/go-playground/validator"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type contextKey string

var (
	contextKeyUserID     = contextKey("UserID")
	contextKeyAuthHeader = contextKey("AuthHeader")
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

func DebugPrintResponseBody(r io.ReadCloser) error {
	if r == nil {
		return errors.New("body is NIL")
	}
	defer r.Close()
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	log.Println(string(body))
	return nil
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
				if GetTeslaAPI().IsKnownAccessToken(h) {
					parsedToken, _ := jwt.Parse(h, nil)
					if !(parsedToken == nil || parsedToken.Claims == nil) {
						exp, err := parsedToken.Claims.GetExpirationTime()
						if err == nil {
							now := time.Now().UTC()
							if exp.After(now) {
								userID, _ = parsedToken.Claims.GetSubject()
								authHeader = h
								tokenValid = true
							}
						}

					}
				}
			}
		}

		if !tokenValid {
			authURLs := []string{
				"/api/1/tesla/",
			}
			url := r.URL.RequestURI()
			for _, authURL := range authURLs {
				authURL = strings.TrimSpace(authURL)
				authURL = strings.TrimSuffix(authURL, "/")
				if authURL != "" && (url == authURL || strings.HasPrefix(url, authURL+"/")) {
					log.Println(authURL + " // " + url)
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

func ServeHTTP() {
	log.Println("Initializing REST services...")

	router := mux.NewRouter()
	routers := make(map[string]Route)
	routers["/api/1/auth/"] = &AuthRouter{}
	routers["/api/1/tesla/"] = &TeslaRouter{}
	routers["/api/1/user/"] = &UserRouter{}

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

	httpServer := &http.Server{
		Addr:         "0.0.0.0:8080",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
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
