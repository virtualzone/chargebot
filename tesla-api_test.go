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

func (a *TeslaAPIMock) IsKnownAccessToken(token string) bool {
	args := a.Called(token)
	return args.Bool(0)
}

func (a *TeslaAPIMock) GetTokens(code string, redirectURI string) (*TeslaAPITokenReponse, error) {
	args := a.Called(code, redirectURI)
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

func (a *TeslaAPIMock) GetOrRefreshAccessToken(userID string) string {
	args := a.Called(userID)
	return args.String(0)
}

func (a *TeslaAPIMock) GetCachedAccessToken(userID string) string {
	args := a.Called(userID)
	return args.String(0)
}

func (a *TeslaAPIMock) InitSession(authToken string, v *Vehicle, wakeUp bool) (*vehicle.Vehicle, error) {
	args := a.Called(authToken, v, wakeUp)
	if resp, ok := args.Get(0).(*vehicle.Vehicle); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) ListVehicles(authToken string) ([]TeslaAPIVehicleEntity, error) {
	args := a.Called(authToken)
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

func (a *TeslaAPIMock) GetVehicleData(authToken string, vehicle *Vehicle) (*TeslaAPIVehicleData, error) {
	args := a.Called(authToken, vehicle)
	if resp, ok := args.Get(0).(*TeslaAPIVehicleData); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) Wakeup(authToken string, vehicle *Vehicle) error {
	args := a.Called(authToken, vehicle)
	return args.Error(0)
}

func UpdateTeslaAPIMockData(api *TeslaAPIMock, vehicleID int, batteryLevel int, chargingState string) {
	vData := &TeslaAPIVehicleData{
		VehicleID: vehicleID,
		ChargeState: TeslaAPIChargeState{
			BatteryLevel:  batteryLevel,
			ChargingState: chargingState,
		},
	}
	api.On("GetVehicleData", "token", mock.Anything).Unset()
	api.On("GetVehicleData", "token", mock.Anything).Return(vData, nil)
}
