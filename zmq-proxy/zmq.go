package main

import (
	"log"
	"os"
	"os/signal"

	"chargebot.io/zmq-proxy/protos"
	zmq "github.com/pebbe/zmq4"
	"google.golang.org/protobuf/proto"
)

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
		log.Println(data)
	}
}
