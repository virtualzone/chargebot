package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/teslamotors/vehicle-command/pkg/protocol"
)

type Config struct {
	ClientID      string
	ClientSecret  string
	Audience      string
	DBFile        string
	Hostname      string
	DevProxy      bool
	Reset         bool
	ManualControl bool
	PrivateKey    protocol.ECDHPrivateKey
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
	c.Hostname = c.getEnv("DOMAIN", "chargebot.io")
	c.DevProxy = (c.getEnv("DEV_PROXY", "0") == "1")
	c.Reset = (c.getEnv("RESET", "0") == "1")
	c.ManualControl = (c.getEnv("MANUAL_CONTROL", "0") == "1")
	privateKeyFile := c.getEnv("PRIVATE_KEY", "./private.key")
	if privateKeyFile != ":none:" {
		privateKey, err := protocol.LoadPrivateKey(privateKeyFile)
		if err != nil {
			log.Panicf("could not load private key: %s\n", err.Error())
		}
		c.PrivateKey = privateKey
	}
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
