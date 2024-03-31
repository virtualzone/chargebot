package main

import (
	"github.com/stretchr/testify/mock"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
	. "github.com/virtualzone/chargebot/goshared"
)

type TeslaAPIMock struct {
	mock.Mock
}

func (a *TeslaAPIMock) GetTokens(audience string, code string, redirectURI string) (*TeslaAPITokenReponse, error) {
	args := a.Called(audience, code, redirectURI)
	if resp, ok := args.Get(0).(*TeslaAPITokenReponse); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) RefreshToken(refreshToken string) (*TeslaAPITokenReponse, error) {
	args := a.Called(refreshToken)
	if resp, ok := args.Get(0).(*TeslaAPITokenReponse); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) InitSession(accessToken string, vin string) (*vehicle.Vehicle, error) {
	args := a.Called(accessToken, vin)
	if resp, ok := args.Get(0).(*vehicle.Vehicle); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) ListVehicles(audience string, accessToken string) ([]TeslaAPIVehicleEntity, error) {
	args := a.Called(audience, accessToken)
	if resp, ok := args.Get(0).([]TeslaAPIVehicleEntity); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) ChargeStart(car *vehicle.Vehicle) error {
	args := a.Called(car)
	return args.Error(0)
}

func (a *TeslaAPIMock) ChargeStop(car *vehicle.Vehicle) error {
	args := a.Called(car)
	return args.Error(0)
}

func (a *TeslaAPIMock) SetChargeLimit(car *vehicle.Vehicle, limitPercent int) error {
	args := a.Called(car, limitPercent)
	return args.Error(0)
}

func (a *TeslaAPIMock) SetChargeAmps(car *vehicle.Vehicle, amps int) error {
	args := a.Called(car, amps)
	return args.Error(0)
}

func (a *TeslaAPIMock) GetVehicleData(audience string, accessToken string, vin string) (*TeslaAPIVehicleData, error) {
	args := a.Called(audience, accessToken, vin)
	if resp, ok := args.Get(0).(*TeslaAPIVehicleData); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) Wakeup(audience string, accessToken string, vin string) error {
	args := a.Called(audience, accessToken, vin)
	return args.Error(0)
}

func (a *TeslaAPIMock) CreateTelemetryConfig(audience string, accessToken string, vin string) error {
	args := a.Called(audience, accessToken, vin)
	return args.Error(0)
}

func (a *TeslaAPIMock) DeleteTelemetryConfig(audience string, accessToken string, vin string) error {
	args := a.Called(audience, accessToken, vin)
	return args.Error(0)
}

func UpdateTeslaAPIMockData(api *TeslaAPIMock, vin string, batteryLevel int, chargingState string) {
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
