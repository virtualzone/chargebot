package main

import (
	"slices"
	"strings"
	"sync"

	. "github.com/virtualzone/chargebot/goshared"
)

type TelemetryQueue struct {
	activeVINsMutex sync.Mutex
	activeVINs      []string
	states          map[string]*PersistedTelemetryState
}

var telemetryQueueInstance *TelemetryQueue = &TelemetryQueue{
	activeVINs: []string{},
	states:     make(map[string]*PersistedTelemetryState),
}

func GetTelemetryQueue() *TelemetryQueue {
	return telemetryQueueInstance
}

func (q *TelemetryQueue) AddActiveVIN(vin string) {
	vin = strings.ToLower(vin)
	if !slices.Contains(q.activeVINs, vin) {
		q.activeVINsMutex.Lock()
		defer q.activeVINsMutex.Unlock()
		q.activeVINs = append(q.activeVINs, vin)
	}
}

func (q *TelemetryQueue) RemoveActiveVIN(vin string) {
	vin = strings.ToLower(vin)
	q.activeVINsMutex.Lock()
	defer q.activeVINsMutex.Unlock()
	q.activeVINs = slices.DeleteFunc(q.activeVINs, func(v string) bool {
		return v == vin
	})
}

func (q *TelemetryQueue) SetState(state *PersistedTelemetryState) {
	q.activeVINsMutex.Lock()
	defer q.activeVINsMutex.Unlock()
	q.states[strings.ToLower(state.VIN)] = state
}

func (q *TelemetryQueue) GetState(vin string) *PersistedTelemetryState {
	state, ok := q.states[strings.ToLower(vin)]
	if !ok {
		return nil
	}
	return state
}
