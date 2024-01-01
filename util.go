package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math/rand"
	"time"
)

func IsCurrentHourUTC(now *time.Time, ts *time.Time) bool {
	if ts.Year() == now.Year() &&
		ts.Month() == now.Month() &&
		ts.Day() == now.Day() &&
		ts.Hour() == now.Hour() {
		return true
	}
	return false
}

func UpdateVehicleDataSaveSoC(authToken string, vehicle *Vehicle) int {
	data, err := GetTeslaAPI().GetVehicleData(authToken, vehicle)
	if err != nil {
		log.Println(err)
		GetDB().LogChargingEvent(vehicle.ID, LogEventVehicleUpdateData, err.Error())
		return 0
	} else {
		GetDB().SetVehicleStateSoC(vehicle.ID, data.ChargeState.BatteryLevel)
		GetDB().LogChargingEvent(vehicle.ID, LogEventVehicleUpdateData, fmt.Sprintf("vehicle SoC updated: %d", data.ChargeState.BatteryLevel))
		return data.ChargeState.BatteryLevel
	}
}

func GetSHA256Hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func IsValidHash(plain string, hash string) bool {
	s := GetSHA256Hash(plain)
	return s == hash
}

func GeneratePassword(length int, includeNumber bool, includeSpecial bool) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var password []byte
	var charSource string

	if includeNumber {
		charSource += "0123456789"
	}
	if includeSpecial {
		charSource += "!@#$%^&*()_+=-"
	}
	charSource += charset

	for i := 0; i < length; i++ {
		randNum := rand.Intn(len(charSource))
		password = append(password, charSource[randNum])
	}
	return string(password)
}
