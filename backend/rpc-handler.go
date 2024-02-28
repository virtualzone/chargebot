package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

func ServeRPC() {
	log.Println("Initializing RPC services...")
	t := new(VehicleStateTelemetry)
	rpc.Register(t)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("listen error:", err)
	}
	go http.Serve(l, nil)
}
