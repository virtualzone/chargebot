package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"slices"
	"strconv"
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

func UpdateVehicleDataSaveSoC(vehicle *Vehicle) (int, *TeslaAPIVehicleData) {
	data, err := GetTeslaAPI().GetVehicleData(vehicle)
	if err != nil {
		log.Println(err)
		GetDB().LogChargingEvent(vehicle.VIN, LogEventVehicleUpdateData, err.Error())
		return 0, nil
	} else {
		GetDB().SetVehicleStateSoC(vehicle.VIN, data.ChargeState.BatteryLevel)
		GetDB().LogChargingEvent(vehicle.VIN, LogEventVehicleUpdateData, fmt.Sprintf("vehicle SoC updated: %d", data.ChargeState.BatteryLevel))
		return data.ChargeState.BatteryLevel, data
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

func RetryHTTPJSONRequest(req *http.Request, authToken string) (*http.Response, error) {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+authToken)
	return RetryHTTPRequest(req)
}

func RetryHTTPRequest(req *http.Request) (*http.Response, error) {
	isRetryCode := func(code int) bool {
		retryCodes := []int{405, 408, 412}
		return slices.Contains(retryCodes, code)
	}

	client := &http.Client{}
	retryCounter := 1
	var resp *http.Response
	var err error
	for retryCounter <= 3 {
		resp, err = client.Do(req)
		if err != nil || (resp != nil && isRetryCode(resp.StatusCode)) {
			time.Sleep(2 * time.Second)
			retryCounter++
		} else {
			retryCounter = 999
		}
	}
	return resp, err
}

func AtoiArray(arr []string) ([]int, error) {
	res := []int{}
	for _, s := range arr {
		i, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		res = append(res, i)
	}
	return res, nil
}

func GetHourstamp(year int, month int, day int, hour int) int {
	hourstamp, _ := strconv.Atoi(fmt.Sprintf("%4d%02d%02d%02d", year, month, day, hour))
	return hourstamp
}

func LogDebug(s string) {
	log.Println("DEBUG: " + s)
}

func OnVehicleUnplugged(vehicle *Vehicle, oldState *VehicleState) {
	// vehicle got plugged out
	GetDB().SetVehicleStatePluggedIn(vehicle.VIN, false)
	GetDB().LogChargingEvent(vehicle.VIN, LogEventVehicleUnplug, "")
	if oldState != nil && oldState.Charging != ChargeStateNotCharging {
		// Vehicle got unplugged while charging
		GetDB().SetVehicleStateCharging(vehicle.VIN, ChargeStateNotCharging)
	}
}

func OnVehiclePluggedIn(vehicle *Vehicle) {
	// vehicle got plugged in at home
	GetDB().SetVehicleStatePluggedIn(vehicle.VIN, true)
	GetDB().LogChargingEvent(vehicle.VIN, LogEventVehiclePlugIn, "")
	if vehicle.Enabled {
		go func() {
			// wait a few moments to ensure vehicle is online
			time.Sleep(10 * time.Second)
			car, err := GetTeslaAPI().InitSession(vehicle, false)
			if err != nil {
				log.Printf("could not init session for vehicle %s on plug in: %s\n", vehicle.VIN, err.Error())
				return
			}
			time.Sleep(5 * time.Second)
			if err := GetTeslaAPI().ChargeStop(car); err != nil {
				log.Printf("could not stop charging for vehicle %s on plug in: %s\n", vehicle.VIN, err.Error())
			}
		}()
	}
}

func IsVehicleHome(telemetryState *TelemetryState, user *User) bool {
	dist := getDistanceFromLatLonInMeters(user.HomeLatitude, user.HomeLongitude, telemetryState.Latitude, telemetryState.Longitude)
	return dist <= user.HomeRadius
}

func CanUpdateVehicleData(vin string, now *time.Time) bool {
	event := GetDB().GetLatestChargingEvent(vin, LogEventVehicleUpdateData)
	if event == nil {
		return true
	}
	limit := now.Add(time.Minute * time.Duration(MaxVehicleDataUpdateIntervalMinutes) * -1)
	return event.Timestamp.Before(limit)
}

func getDistanceFromLatLonInMeters(lat1 float64, lon1 float64, lat2 float64, lon2 float64) int {
	r := 6371 * 1000.0           // Radius of the earth in meters
	dLat := deg2rad(lat2 - lat1) // deg2rad below
	dLon := deg2rad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(deg2rad(lat1))*math.Cos(deg2rad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := r * c // Distance in meters
	return int(d)
}

func deg2rad(deg float64) float64 {
	return deg * (math.Pi / 180)
}
