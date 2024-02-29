package main

import (
	"github.com/stretchr/testify/mock"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
)

type TeslaAPIMock struct {
	mock.Mock
}

func (a *TeslaAPIMock) InitTokenCache() {
	a.Called()
}

func (a *TeslaAPIMock) GetTokens(userID string, code string, redirectURI string) (*TeslaAPITokenReponse, error) {
	args := a.Called(userID, code, redirectURI)
	if resp, ok := args.Get(0).(*TeslaAPITokenReponse); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) RefreshToken(userID string, refreshToken string) (*TeslaAPITokenReponse, error) {
	args := a.Called(userID, refreshToken)
	if resp, ok := args.Get(0).(*TeslaAPITokenReponse); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) GetOrRefreshAccessToken(userID string) string {
	args := a.Called(userID)
	return args.String(0)
}

func (a *TeslaAPIMock) GetCachedAccessToken(userID string) string {
	args := a.Called(userID)
	return args.String(0)
}

func (a *TeslaAPIMock) InitSession(v *Vehicle, wakeUp bool) (*vehicle.Vehicle, error) {
	args := a.Called(v, wakeUp)
	if resp, ok := args.Get(0).(*vehicle.Vehicle); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) ListVehicles(userID string) ([]TeslaAPIVehicleEntity, error) {
	args := a.Called(userID)
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

func (a *TeslaAPIMock) GetVehicleData(vehicle *Vehicle) (*TeslaAPIVehicleData, error) {
	args := a.Called(vehicle)
	if resp, ok := args.Get(0).(*TeslaAPIVehicleData); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) Wakeup(vehicle *Vehicle) error {
	args := a.Called(vehicle)
	return args.Error(0)
}

func (a *TeslaAPIMock) CreateTelemetryConfig(vehicle *Vehicle) error {
	args := a.Called(vehicle)
	return args.Error(0)
}

func (a *TeslaAPIMock) DeleteTelemetryConfig(vehicle *Vehicle) error {
	args := a.Called(vehicle)
	return args.Error(0)
}

func UpdateTeslaAPIMockData(api *TeslaAPIMock, vehicleID int, batteryLevel int, chargingState string) {
	GetDB().SetVehicleStateSoC(vehicleID, batteryLevel)
	//GetDB().SetVehicleStateCharging(vehicleID, chargingState)
	vData := &TeslaAPIVehicleData{
		VehicleID: vehicleID,
		ChargeState: TeslaAPIChargeState{
			BatteryLevel:  batteryLevel,
			ChargingState: chargingState,
		},
	}
	api.On("GetVehicleData", mock.Anything).Unset()
	api.On("GetVehicleData", mock.Anything).Return(vData, nil)
}
