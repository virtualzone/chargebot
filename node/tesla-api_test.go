package main

import (
	"github.com/stretchr/testify/mock"
	. "github.com/virtualzone/chargebot/goshared"
)

type TeslaAPIMock struct {
	mock.Mock
}

func (a *TeslaAPIMock) RefreshToken(refreshToken string) (*TeslaAPITokenReponse, error) {
	args := a.Called(refreshToken)
	if resp, ok := args.Get(0).(*TeslaAPITokenReponse); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

/*
func (a *TeslaAPIMock) GetOrRefreshAccessToken(userID string) string {
	args := a.Called(userID)
	return args.String(0)
}

func (a *TeslaAPIMock) GetCachedAccessToken(userID string) string {
	args := a.Called(userID)
	return args.String(0)
}
*/

func (a *TeslaAPIMock) ListVehicles() ([]TeslaAPIVehicleEntity, error) {
	args := a.Called()
	if resp, ok := args.Get(0).([]TeslaAPIVehicleEntity); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) ChargeStart(vin string) error {
	args := a.Called(vin)
	return args.Error(0)
}

func (a *TeslaAPIMock) ChargeStop(vin string) error {
	args := a.Called(vin)
	return args.Error(0)
}

func (a *TeslaAPIMock) SetChargeLimit(vin string, limitPercent int) error {
	args := a.Called(vin, limitPercent)
	return args.Error(0)
}

func (a *TeslaAPIMock) SetChargeAmps(vin string, amps int) error {
	args := a.Called(vin, amps)
	return args.Error(0)
}

func (a *TeslaAPIMock) GetVehicleData(vin string) (*TeslaAPIVehicleData, error) {
	args := a.Called(vin)
	if resp, ok := args.Get(0).(*TeslaAPIVehicleData); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) Wakeup(vin string) error {
	args := a.Called(vin)
	return args.Error(0)
}

func (a *TeslaAPIMock) CreateTelemetryConfig(vin string) error {
	args := a.Called(vin)
	return args.Error(0)
}

func (a *TeslaAPIMock) DeleteTelemetryConfig(vin string) error {
	args := a.Called(vin)
	return args.Error(0)
}

func (a *TeslaAPIMock) RegisterVehicle(vin string) error {
	args := a.Called(vin)
	return args.Error(0)
}

func (a *TeslaAPIMock) UnregisterVehicle(vin string) error {
	args := a.Called(vin)
	return args.Error(0)
}

func (a *TeslaAPIMock) GetTelemetryState(vin string) (*PersistedTelemetryState, error) {
	args := a.Called(vin)
	if resp, ok := args.Get(0).(*PersistedTelemetryState); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func UpdateTeslaAPIMockData(api *TeslaAPIMock, vin string, batteryLevel int, chargingState string) {
	GetDB().SetVehicleStateSoC(vin, batteryLevel)
	vData := &TeslaAPIVehicleData{
		VIN: vin,
		ChargeState: TeslaAPIChargeState{
			BatteryLevel:  batteryLevel,
			ChargingState: chargingState,
		},
	}
	api.On("GetVehicleData", mock.Anything).Unset()
	api.On("GetVehicleData", mock.Anything).Return(vData, nil)
}
