package main

import (
	. "github.com/virtualzone/chargebot/goshared"
)

type TeslaAPI interface {
	RefreshToken(refreshToken string) (*TeslaAPITokenReponse, error)
	ListVehicles() ([]TeslaAPIVehicleEntity, error)
	ChargeStart(vin string) error
	ChargeStop(vin string) error
	SetChargeLimit(vin string, limitPercent int) error
	SetChargeAmps(vin string, amps int) error
	GetVehicleData(vin string) (*TeslaAPIVehicleData, error)
	Wakeup(vin string) error
	CreateTelemetryConfig(vin string) error
	DeleteTelemetryConfig(vin string) error
	RegisterVehicle(vin string) error
	UnregisterVehicle(vin string) error
	GetTelemetryState(vin string) (*PersistedTelemetryState, error)
}
