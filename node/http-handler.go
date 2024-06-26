package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

var httpRouter *mux.Router

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

func InitHTTPRouter() {
	router := mux.NewRouter()
	routers := make(map[string]Route)
	routers["/api/1/ctrl/"] = &ManualControlRouter{}
	routers["/api/1/tesla/"] = &TeslaRouter{}
	routers["/api/1/user/"] = &UserRouter{}

	for prefix, route := range routers {
		subRouter := router.PathPrefix(prefix).Subrouter()
		route.SetupRoutes(subRouter)
	}

	if GetConfig().DevProxy {
		target, _ := url.Parse("http://localhost:3001")
		proxy := httputil.NewSingleHostReverseProxy(target)
		router.PathPrefix("/").Handler(proxy)
	} else {
		fs := http.FileServer(http.Dir("./static"))
		router.PathPrefix("/").Handler(fs)
	}

	httpRouter = router
}

func ServeHTTP() {
	log.Println("Initializing REST services...")
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", GetConfig().Port),
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
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		httpServer.Shutdown(ctx)
	}()
}
