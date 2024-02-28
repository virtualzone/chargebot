package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	zmq "github.com/pebbe/zmq4"
)

func ServeZMQ() {
	if GetConfig().ZMQPublisher == "" {
		return
	}

	log.Println("Initializing ZMQ subscriber...")

	zctx, _ := zmq.NewContext()
	s, _ := zctx.NewSocket(zmq.SUB)
	//defer s.Close()
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
			address, err := s.Recv(0)
			if err != nil {
				panic(err)
			}

			if msg, err := s.Recv(0); err != nil {
				panic(err)
			} else {
				fmt.Printf("ZMQ message in channel %s: %s\n", address, msg)
			}
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	s.Close()
}
