package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/teslamotors/vehicle-command/pkg/protocol"
)

type Config struct {
	TeslaClientID      string
	TeslaClientSecret  string
	TeslaAudience      string
	TeslaTelemetryHost string
	TeslaTelemetryCA   string
	DBFile             string
	Hostname           string
	DevProxy           bool
	Reset              bool
	TeslaPrivateKey    protocol.ECDHPrivateKey
	ZMQPublisher       string
	DebugLog           bool
	CryptKey           string
	AuthURL            string
	AuthClientID       string
	AuthClientSecret   string
	AuthRolesPath      string
	AuthFieldEmail     string
	AuthFieldUsername  string
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
	c.TeslaClientID = c.getEnv("TESLA_CLIENT_ID", "e9941f08e0d6-4c2f-b8ee-291060ec648a")
	c.TeslaClientSecret = c.getEnv("TESLA_CLIENT_SECRET", "")
	c.TeslaAudience = c.getEnv("TESLA_AUDIENCE", "https://fleet-api.prd.eu.vn.cloud.tesla.com")
	c.TeslaTelemetryHost = c.getEnv("TESLA_TELEMETRY_HOST", "tesla-telemetry.chargebot.io")
	c.TeslaTelemetryCA = c.getEnv("TESLA_TELEMETRY_CA", "")
	if c.TeslaTelemetryCA != "" {
		ca, err := os.ReadFile(c.TeslaTelemetryCA)
		if err != nil {
			log.Panicf("could not load ca file: %s\n", err.Error())
		}
		c.TeslaTelemetryCA = strings.ReplaceAll(string(ca), "\r", "")
	}
	c.DBFile = c.getEnv("DB_FILE", "/tmp/tgc.db")
	c.Hostname = c.getEnv("DOMAIN", "chargebot.io")
	c.DevProxy = (c.getEnv("DEV_PROXY", "0") == "1")
	c.Reset = (c.getEnv("RESET", "0") == "1")
	privateKeyFile := c.getEnv("TESLA_PRIVATE_KEY", "./private.key")
	if privateKeyFile != ":none:" {
		privateKey, err := protocol.LoadPrivateKey(privateKeyFile)
		if err != nil {
			log.Panicf("could not load private key: %s\n", err.Error())
		}
		c.TeslaPrivateKey = privateKey
	}
	c.ZMQPublisher = c.getEnv("ZMQ_PUB", "")
	c.DebugLog = (c.getEnv("DEBUG_LOG", "0") == "1")
	c.CryptKey = c.getEnv("CRYPT_KEY", "")
	if len(c.CryptKey) != 32 {
		log.Panicln("CRYPT_KEY must be 32 bytes long")
	}
	c.AuthURL = c.getEnv("AUTH_URL", "https://auth.chargebot.io/realms/chargebot")
	c.AuthClientID = c.getEnv("AUTH_CLIENT_ID", "chargebot.io-website")
	c.AuthClientSecret = c.getEnv("AUTH_CLIENT_SECRET", "")
	c.AuthRolesPath = c.getEnv("AUTH_ROLES_PATH", "resource_access.portfolio-test.roles")
	c.AuthFieldEmail = c.getEnv("AUTH_FIELD_EMAIL", "email")
	c.AuthFieldUsername = c.getEnv("AUTH_FIELD_USERNAME", "preferred_username")
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
