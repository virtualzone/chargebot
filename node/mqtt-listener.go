package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttSubscriber struct {
	Interrupt chan os.Signal
}

func (m *MqttSubscriber) connectHandler(client mqtt.Client) {
	if token := client.Subscribe(GetConfig().MqttTopicSurplus, 0, m.messageSurplusHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
}

func (m *MqttSubscriber) connectionLostHandler(client mqtt.Client, err error) {
}

func (m *MqttSubscriber) messageSurplusHandler(client mqtt.Client, msg mqtt.Message) {
	go func() {
		surplus, err := strconv.Atoi(string(msg.Payload()))
		if err != nil {
			log.Printf("Could not parse surplus to int: %s\n", msg.Payload())
			return
		}
		GetDB().RecordSurplus(surplus)
	}()
}

func (m *MqttSubscriber) Listen() {
	if GetConfig().MqttBroker == "" {
		return
	}

	log.Println("Initializing MQTT subscriber...")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(GetConfig().MqttBroker)
	opts.SetClientID(GetConfig().MqttClientID)
	opts.SetUsername(GetConfig().MqttUsername)
	opts.SetPassword(GetConfig().MqttPassword)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.OnConnect = m.connectHandler
	opts.OnConnectionLost = m.connectionLostHandler

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	go func() {
		for {
			select {
			case <-m.Interrupt:
				if token := c.Unsubscribe(GetConfig().MqttTopicSurplus); token.Wait() && token.Error() != nil {
					fmt.Println(token.Error())
					os.Exit(1)
				}

				c.Disconnect(250)
			}
		}
	}()
}
