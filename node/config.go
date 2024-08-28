package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/teslamotors/vehicle-command/pkg/protocol"
)

type Config struct {
	TeslaClientID          string
	TeslaRefreshToken      string
	DBFile                 string
	Port                   int
	Token                  string
	TokenPassword          string
	TelemetryEndpoint      string
	CmdEndpoint            string
	DevProxy               bool
	CryptKey               string
	TelegramToken          string
	TelegramChatID         string
	PlugStateAutodetection bool
	InitDBOnly             bool
	DemoMode               bool
	MqttBroker             string
	MqttClientID           string
	MqttUsername           string
	MqttPassword           string
	MqttTopicSurplus       string
	BLE                    bool
	TeslaPrivateKey        protocol.ECDHPrivateKey
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
	c.TeslaRefreshToken = c.getEnv("TESLA_REFRESH_TOKEN", "")
	c.DBFile = c.getEnv("DB_FILE", "/tmp/chargebot_node.db")
	port, err := strconv.Atoi(c.getEnv("PORT", "8080"))
	if err != nil {
		log.Panicln("PORT must be numeric")
	}
	c.Port = port
	c.Token = c.getEnv("TOKEN", "")
	c.TokenPassword = c.getEnv("PASSWORD", "")
	c.TelemetryEndpoint = c.getEnv("TELEMETRY_ENDPOINT", "wss://chargebot.io/api/1/user/{token}/ws")
	c.CmdEndpoint = c.getEnv("CMD_ENDPOINT", "https://chargebot.io/api/1/user/{token}")
	c.TelemetryEndpoint = strings.ReplaceAll(c.TelemetryEndpoint, "{token}", c.Token)
	c.CmdEndpoint = strings.ReplaceAll(c.CmdEndpoint, "{token}", c.Token)
	c.DevProxy = (c.getEnv("DEV_PROXY", "0") == "1")
	c.CryptKey = c.getEnv("CRYPT_KEY", "")
	c.TelegramToken = c.getEnv("TELEGRAM_TOKEN", "")
	c.TelegramChatID = c.getEnv("TELEGRAM_CHAT_ID", "")
	c.PlugStateAutodetection = (c.getEnv("PLUG_AUTODETECT", "1") == "1")
	c.InitDBOnly = (c.getEnv("INIT_DB_ONLY", "0") == "1")
	c.DemoMode = (c.getEnv("DEMO_MODE", "0") == "1")
	c.MqttBroker = c.getEnv("MQTT_BROKER", "")
	c.MqttClientID = c.getEnv("MQTT_CLIENT_ID", "chargebot")
	c.MqttUsername = c.getEnv("MQTT_USERNAME", "")
	c.MqttPassword = c.getEnv("MQTT_PASSWORD", "")
	c.MqttTopicSurplus = c.getEnv("MQTT_TOPIC_SURPLUS", "chargebot/surplus")
	c.BLE = (c.getEnv("BLE", "0") == "1")

	privateKeyFile := c.getEnv("TESLA_PRIVATE_KEY", "")
	if c.BLE && privateKeyFile == "" {
		log.Panicf("need to specify TESLA_PRIVATE_KEY when using BLE connection\n")
	}
	if privateKeyFile != "" {
		privateKey, err := protocol.LoadPrivateKey(privateKeyFile)
		if err != nil {
			log.Panicf("could not load private key: %s\n", err.Error())
		}
		c.TeslaPrivateKey = privateKey
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
