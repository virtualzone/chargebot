package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type Config struct {
	ClientID     string
	ClientSecret string
	Audience     string
	DBFile       string
	Hostname     string
	DevProxy     bool
	Reset        bool
}

var _configInstance *Config
var _configOnce sync.Once

func GetConfig() *Config {
	_configOnce.Do(func() {
		_configInstance = &Config{}
		_configInstance.ReadConfig()
	})
	return _configInstance
}

func (c *Config) ReadConfig() {
	c.ClientID = c.getEnv("CLIENT_ID", "e9941f08e0d6-4c2f-b8ee-291060ec648a")
	c.ClientSecret = c.getEnv("CLIENT_SECRET", "")
	c.Audience = c.getEnv("AUDIENCE", "https://fleet-api.prd.eu.vn.cloud.tesla.com")
	c.DBFile = c.getEnv("DB_FILE", "/tmp/tgc.db")
	c.Hostname = c.getEnv("DOMAIN", "tgc.virtualzone.de")
	c.DevProxy = (c.getEnv("DEV_PROXY", "0") == "1")
	c.Reset = (c.getEnv("RESET", "0") == "1")
}

func (c *Config) Print() {
	s, _ := json.MarshalIndent(c, "", "\t")
	log.Println("Using config:\n" + string(s))
}

func (c *Config) getEnv(key, defaultValue string) string {
	res := os.Getenv(key)
	if res == "" {
		return defaultValue
	}
	return res
}
