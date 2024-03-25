package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	. "github.com/virtualzone/chargebot/goshared"
)

type TelemetryPoller struct {
	Interrupt chan os.Signal
	ticker    *time.Ticker
	vins      []string
	lastKnown map[string]int64
	polling   bool
}

func (t *TelemetryPoller) Poll() {
	t.initVins()
	t.polling = false
	t.ticker = time.NewTicker(time.Second * 5)
	go func() {
		for {
			select {
			case <-t.ticker.C:
				if t.polling {
					break
				}
				t.polling = true
				for _, vin := range t.vins {
					t.getVehicleState(vin)
				}
				t.polling = false
			case <-t.Interrupt:
				return
			}
		}
	}()
}

func (t *TelemetryPoller) Reconnect() {
	t.initVins()
}

func (t *TelemetryPoller) getVehicleState(vin string) {
	state, err := GetTeslaAPI().GetTelemetryState(vin)
	if err != nil {
		log.Println(err)
		return
	}
	if state == nil {
		return
	}
	lastKnown, ok := t.lastKnown[vin]
	if !ok || lastKnown < state.UTC {
		t.processState(state)
		t.lastKnown[vin] = state.UTC
	}
}

func (t *TelemetryPoller) initVins() {
	t.vins = []string{}
	t.lastKnown = make(map[string]int64)
	vehicles := GetDB().GetVehicles()
	for _, v := range vehicles {
		t.vins = append(t.vins, v.VIN)
		t.lastKnown[v.VIN] = 0
	}
}

/*
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

func (t *TelemetryPoller) Reconnect() {
	log.Println("Reconnecting Telemetry Websocket...")
	t.Interrupt <- os.Interrupt
	time.Sleep(2 * time.Second)
	t.Poll()
}

func (t *TelemetryPoller) sendAuth(c *websocket.Conn) error {
	payload := PasswordProtectedRequest{
		Password: GetConfig().TokenPassword,
	}
	json, _ := json.Marshal(payload)
	if err := c.WriteMessage(websocket.TextMessage, json); err != nil {
		return err
	}
	return nil
}

func (t *TelemetryPoller) connectAndListen() bool {
	u, err := url.Parse(GetConfig().TelemetryEndpoint)
	if err != nil {
		log.Fatal("url:", err)
	}
	log.Print("Using websocket host: ", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Print("Error dialing websocket:", err)
		return false
	}
	defer c.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Error reading from websocket:", err)
				return
			}
			if mt != websocket.TextMessage {
				continue
			}
			text := string(message)
			if text == "false" {
				log.Println("Websocket authentication failed")
			} else if text == "true" {
				log.Println("Websocket authentication successful")
			} else if strings.Index(text, "{") == 0 {
				var m PersistedTelemetryState
				if err = json.Unmarshal(message, &m); err != nil {
					log.Println("Error unmarshalling telemetry message:", err)
					continue
				}
				t.processState(&m)
			}
		}
	}()

	if err := t.sendAuth(c); err != nil {
		log.Println("Error sending websocket auth:", err)
		return false
	}

	for {
		select {
		case <-done:
			return false
		case <-t.Interrupt:
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Error writing websocket close:", err)
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
*/

func (t *TelemetryPoller) processState(telemetryState *PersistedTelemetryState) {
	vehicle := GetDB().GetVehicleByVIN(telemetryState.VIN)
	if vehicle == nil {
		log.Printf("could not find vehicle by vin for telemetry data: %s\n", telemetryState.VIN)
		return
	}

	sState, _ := json.Marshal(telemetryState)
	LogDebug("Processing vehicle state: " + string(sState))

	oldState := GetDB().GetVehicleState(vehicle.VIN)
	if oldState == nil {
		oldState = &VehicleState{
			PluggedIn: false,
			Charging:  ChargeStateNotCharging,
			Amps:      -1,
			SoC:       -1,
		}
	}

	if oldState.Amps != telemetryState.Amps {
		GetDB().SetVehicleStateAmps(vehicle.VIN, telemetryState.Amps)
	}
	if oldState.SoC != telemetryState.SoC {
		GetDB().SetVehicleStateSoC(vehicle.VIN, telemetryState.SoC)
	}
	if oldState.ChargeLimit != telemetryState.ChargeLimit {
		GetDB().SetVehicleStateChargeLimit(vehicle.VIN, telemetryState.ChargeLimit)
	}
	if oldState.Charging != ChargeStateNotCharging && !telemetryState.Charging {
		// Only change if charging was not recently started
		event := GetDB().GetLatestChargingEvent(vehicle.VIN, LogEventChargeStart)
		now := time.Now().UTC()
		if event.Timestamp.Before(now.Add(-5 * time.Minute)) {
			GetDB().SetVehicleStateCharging(vehicle.VIN, ChargeStateNotCharging)
		}
	}
	if oldState.IsHome != telemetryState.IsHome {
		GetDB().SetVehicleStateIsHome(vehicle.VIN, telemetryState.IsHome)
	}

	if vehicle.Enabled && oldState.Charging == ChargeStateNotCharging && telemetryState.Charging {
		// if vehicle is charging although assumed not to, it could be that it has been plugged in recently
		if !oldState.PluggedIn && telemetryState.IsHome {
			if GetConfig().PlugStateAutodetection {
				OnVehiclePluggedIn(vehicle)
				return
			}
		} else {
			// otherwise, this is an anomaly where chargebot stopped charging but vehicle is still charging
			// check if charging was actually stopped within the last minutes (else, it might just be the A/C)
			event := GetDB().GetLatestChargingEvent(vehicle.VIN, LogEventChargeStop)
			now := time.Now().UTC()
			if event.Timestamp.After(now.Add(-5 * time.Minute)) {
				log.Printf("Anomaly detected: Vehicle %s was assumed to be not charging, but actually is - stopping it\n", vehicle.VIN)
				GetChargeController().stopCharging(vehicle)
			}
		}
	}

	if GetConfig().PlugStateAutodetection {
		// Workarounds for incorrect ChargeState in telemetry data
		// https://github.com/teslamotors/fleet-telemetry/issues/123
		if oldState.PluggedIn && !telemetryState.IsHome {
			// If vehicle was plugged in but is not home anymore, it is obiously not plugged in anymore
			OnVehicleUnplugged(vehicle, oldState)
			return
		}
		if !telemetryState.IsHome {
			return
		}
		now := time.Now().UTC()
		if CanUpdateVehicleData(vehicle.VIN, &now) {
			data, err := GetTeslaAPI().GetVehicleData(vehicle.VIN)
			if err != nil {
				log.Println(err)
				return
			}
			GetDB().LogChargingEvent(vehicle.VIN, LogEventVehicleUpdateData, "")
			cableConnected := (strings.ToLower(data.ChargeState.ConnectedChargeCable) == "iec" || strings.ToLower(data.ChargeState.ConnectedChargeCable) == "sae")
			if oldState.PluggedIn && !cableConnected {
				OnVehicleUnplugged(vehicle, oldState)
			}
			if !oldState.PluggedIn && cableConnected {
				OnVehiclePluggedIn(vehicle)
			}
		}
	}
	/*
		if oldState.PluggedIn && !telemetryState.PluggedIn {
			t.onVehicleUnplugged(vehicle, oldState)
		}
		if t.isVehicleHome(telemetryState, user) && telemetryState.PluggedIn && !oldState.PluggedIn {
			t.onVehiclePluggedIn(vehicle)
		}
	*/
}
