package main

import (
	"log"
	"net/rpc"
	"os"
	"os/signal"
	"strings"

	"chargebot.io/zmq-proxy/protos"
	zmq "github.com/pebbe/zmq4"
	"google.golang.org/protobuf/proto"
)

type TelemetryState struct {
	VIN         string
	PluggedIn   bool
	Charging    bool
	ChargeLimit int
	SoC         int
	Amps        int
	Latitude    float64
	Longitude   float64
}

func ServeZMQ() {
	if GetConfig().ZMQPublisher == "" {
		return
	}

	log.Println("Initializing ZMQ subscriber...")

	zctx, _ := zmq.NewContext()
	s, _ := zctx.NewSocket(zmq.SUB)
	defer s.Close()
	if err := s.Connect(GetConfig().ZMQPublisher); err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
	if err := s.SetSubscribe(""); err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}

	go func() {
		for {
			zmqLoop(s)
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func zmqLoop(s *zmq.Socket) {
	address, err := s.Recv(0)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("reading zmq message in channel %s\n", address)

	if msg, err := s.RecvBytes(0); err != nil {
		log.Println(err)
		return
	} else {
		data := &protos.Payload{}
		if err := proto.Unmarshal(msg, data); err != nil {
			log.Println(err)
			return
		}
		res := &TelemetryState{
			VIN:         data.Vin,
			PluggedIn:   false,
			Charging:    false,
			ChargeLimit: 0,
			SoC:         0,
			Amps:        0,
			Latitude:    0,
			Longitude:   0,
		}
		for _, e := range data.Data {
			switch e.Key {
			case protos.Field_ChargeAmps:
				res.Amps = int(e.Value.GetIntValue())
			case protos.Field_ChargeLimitSoc:
				res.ChargeLimit = int(e.Value.GetIntValue())
			case protos.Field_Soc:
				res.SoC = int(e.Value.GetFloatValue())
			case protos.Field_ChargeState:
				if strings.ToLower(e.Value.String()) == "idle" {
					res.PluggedIn = true
				} else if strings.ToLower(e.Value.String()) == "enable" {
					res.PluggedIn = true
					res.Charging = true
				}
			case protos.Field_Location:
				res.Latitude = e.Value.GetLocationValue().Latitude
				res.Longitude = e.Value.GetLocationValue().Longitude
			}
		}
		client, err := rpc.DialHTTP("tcp", GetConfig().BackendRPC)
		if err != nil {
			log.Println(err)
			return
		}
		var reply bool
		if err := client.Call("VehicleStateTelemetry.Update", res, &reply); err != nil {
			log.Println(err)
			return
		}
		log.Println(data)
		log.Println(res)
		log.Println(reply)
	}
}
