package main

import (
	"os"
	"testing"
	"time"
)

type MockTime struct {
	CurTime time.Time
}

func (m MockTime) UTCNow() time.Time {
	return m.CurTime
}

var GlobalMockTime *MockTime

func TestMain(m *testing.M) {
	os.Setenv("DB_FILE", ":memory:")
	GetConfig().ReadConfig()
	GlobalMockTime = &MockTime{
		CurTime: time.Now().UTC(),
	}
	GetDB().Time = GlobalMockTime
	GetDB().Connect()
	ResetTestDB()
	code := m.Run()
	os.Exit(code)
}

func ResetTestDB() {
	GetDB().ResetDBStructure()
	GetDB().InitDBStructure()
	TeslaAPIInstance = &TeslaAPIMock{}
	GlobalMockTime.CurTime = time.Now().UTC()
	//TeslaAPIInstance.InitTokenCache()
}

func NewTestChargeController() *ChargeController {
	cc := new(ChargeController)
	cc.Time = GlobalMockTime
	return cc
}

func SetTibberTestPrice(vehicleID int, ts time.Time, price float32) {
	GetDB().SetTibberPrice(vehicleID, ts.Year(), int(ts.Month()), ts.Day(), ts.Hour(), price)
}
