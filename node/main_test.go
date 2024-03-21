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
	os.Setenv("CRYPT_KEY", "12345678901234567890123456789012")
	GetConfig().ReadConfig()
	DelayBetweenAPICommands = time.Second * 0
	GlobalMockTime = &MockTime{
		CurTime: time.Now().UTC(),
	}
	GetDB().Time = GlobalMockTime
	GetDB().Connect()
	ResetTestDB()
	InitHTTPRouter()
	code := m.Run()
	os.Exit(code)
}

func ResetTestDB() {
	GetDB().ResetDBStructure()
	GetDB().InitDBStructure()
	TeslaAPIInstance = &TeslaAPIMock{}
	GlobalMockTime.CurTime = time.Now().UTC()
}

func GetNextMondayMidnight() time.Time {
	now := time.Now().UTC()
	curWeekday := now.Weekday()
	if curWeekday == time.Sunday {
		curWeekday = 7
	}
	now = now.AddDate(0, 0, 8-int(curWeekday))
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return now
}

func SetTibberTestPrice(vin string, ts time.Time, price float32) {
	GetDB().SetTibberPrice(vin, ts.Year(), int(ts.Month()), ts.Day(), ts.Hour(), price)
}

func NewTestChargeController() *ChargeController {
	cc := new(ChargeController)
	cc.Time = GlobalMockTime
	cc.Async = false
	return cc
}
