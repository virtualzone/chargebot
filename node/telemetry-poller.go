package main

import (
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type TelemetryPoller struct {
	Interrupt chan os.Signal
}

func (t *TelemetryPoller) Poll() {
	go func() {
		interrupted := false
		for {
			interrupted = t.connectAndListen()
			if interrupted {
				return
			}
			// delay until next connection attempt
			time.Sleep(5 * time.Second)
		}
	}()
}

func (t *TelemetryPoller) connectAndListen() bool {
	u, err := url.Parse(GetConfig().TelemetryEndpoint)
	if err != nil {
		log.Fatal("url:", err)
	}
	log.Print("using websocket host: ", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Print("dial:", err)
		return false
	}
	defer c.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s, type: %v", message, mt)
		}
	}()

	// TEST
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return false
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return false
			}
		case <-t.Interrupt:
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return true
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return true
		}

	}
}
