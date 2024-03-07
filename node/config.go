package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
)

type Config struct {
	Token             string
	TelemetryEndpoint string
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
	c.Token = c.getEnv("TOKEN", "")
	c.TelemetryEndpoint = c.getEnv("TELEMETRY_ENDPOINT", "wss://chargebot.io/api/1/user/{token}/ws")
	c.TelemetryEndpoint = strings.ReplaceAll(c.TelemetryEndpoint, "{token}", c.Token)
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
