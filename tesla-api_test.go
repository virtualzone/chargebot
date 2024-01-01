package main

import (
	"testing"

	"github.com/stretchr/testify/mock"
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

func (a *TeslaAPIMock) ListVehicles(authToken string) ([]TeslaAPIVehicleEntity, error) {
	args := a.Called(authToken)
	if resp, ok := args.Get(0).([]TeslaAPIVehicleEntity); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) ChargeStart(authToken string, vehicle *Vehicle) (bool, error) {
	args := a.Called(authToken, vehicle)
	return args.Bool(0), args.Error(1)
}

func (a *TeslaAPIMock) ChargeStop(authToken string, vehicle *Vehicle) (bool, error) {
	args := a.Called(authToken, vehicle)
	return args.Bool(0), args.Error(1)
}

func (a *TeslaAPIMock) SetChargeLimit(authToken string, vehicle *Vehicle, limitPercent int) (bool, error) {
	args := a.Called(authToken, vehicle, limitPercent)
	return args.Bool(0), args.Error(1)
}

func (a *TeslaAPIMock) SetChargeAmps(authToken string, vehicle *Vehicle, amps int) (bool, error) {
	args := a.Called(authToken, vehicle, amps)
	return args.Bool(0), args.Error(1)
}

func (a *TeslaAPIMock) GetVehicleData(authToken string, vehicle *Vehicle) (*TeslaAPIVehicleData, error) {
	args := a.Called(authToken, vehicle)
	if resp, ok := args.Get(0).(*TeslaAPIVehicleData); !ok {
		panic("assert: arguments wasn't correct type")
	} else {
		return resp, args.Error(1)
	}
}

func (a *TeslaAPIMock) WakeUpVehicle(authToken string, vehicle *Vehicle) error {
	args := a.Called(authToken, vehicle)
	return args.Error(0)
}

func (a *TeslaAPIMock) SetScheduledCharging(authToken string, vehicle *Vehicle, enable bool, minutesAfterMidnight int) (bool, error) {
	args := a.Called(authToken, vehicle, enable, minutesAfterMidnight)
	return args.Bool(0), args.Error(1)
}

func TestBlah(t *testing.T) {
	teslaAPI := new(TeslaAPIMock)
	//teslaAPI.On("SetChargeAmps", "test", nil, 16) //.Return(true, nil)
	teslaAPI.On("GetCachedAccessToken", "test").Return("ok")
	//teslaAPI.On("SetChargeAmps", mock.Anything, mock.Anything, 32).Return(false, errors.New("geht net!"))

	teslaAPI.GetCachedAccessToken("test")

	//teslaAPI.SetChargeAmps("", nil, 32)

	//teslaAPI.AssertExpectations(t)
}
