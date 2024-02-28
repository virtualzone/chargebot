package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type Config struct {
	ZMQPublisher string
	BackendRPC   string
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
	c.ZMQPublisher = c.getEnv("ZMQ_PUB", "")
	c.BackendRPC = c.getEnv("BACKEND_RPC", "127.0.0.1:1234")
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
