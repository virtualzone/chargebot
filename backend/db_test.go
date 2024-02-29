package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDB_SetVehicleStates(t *testing.T) {
	t.Cleanup(ResetTestDB)

	GetDB().SetVehicleStateCharging(123, ChargeStateChargingOnSolar)
	GetDB().SetVehicleStateAmps(123, 5)
	GetDB().SetVehicleStatePluggedIn(123, true)
	GetDB().SetVehicleStateSoC(123, 55)

	state := GetDB().GetVehicleState(123)
	assert.NotNil(t, state)
	assert.Equal(t, ChargeStateChargingOnSolar, state.Charging)
	assert.Equal(t, 5, state.Amps)
	assert.Equal(t, 55, state.SoC)
	assert.Equal(t, true, state.PluggedIn)

}

func TestDB_CRUDAuthCode(t *testing.T) {
	t.Cleanup(ResetTestDB)

	authCode := GetDB().CreateAuthCode()
	assert.True(t, GetDB().IsValidAuthCode(authCode))
	assert.False(t, GetDB().IsValidAuthCode(authCode+"123"))

	GetDB().DeleteAuthCode(authCode)
	assert.False(t, GetDB().IsValidAuthCode(authCode))
}

func TestDB_CRUDUser(t *testing.T) {
	t.Cleanup(ResetTestDB)

	user := &User{
		ID: "123",
	}
	GetDB().CreateUpdateUser(user)

	user2 := GetDB().GetUser(user.ID)
	assert.NotNil(t, user2)
	assert.EqualValues(t, user, user2)

	GetDB().CreateUpdateUser(user)

	user2 = GetDB().GetUser(user.ID)
	assert.NotNil(t, user2)
	assert.EqualValues(t, user, user2)
}

func TestDB_CRUDVehicle(t *testing.T) {
	t.Cleanup(ResetTestDB)

	vehicle := &Vehicle{
		ID:              123,
		UserID:          "456",
		VIN:             "789",
		DisplayName:     "Model S",
		APIToken:        "",
		Enabled:         true,
		TargetSoC:       65,
		MaxAmps:         8,
		NumPhases:       3,
		SurplusCharging: true,
		MinSurplus:      1250,
		MinChargeTime:   25,
		LowcostCharging: true,
		MaxPrice:        22,
		GridProvider:    "tibber",
		GridStrategy:    2,
		DepartDays:      "1357",
		DepartTime:      "07:15:00",
		TibberToken:     "def",
	}
	GetDB().CreateUpdateVehicle(vehicle)

	vehicle2 := GetDB().GetVehicleByID(vehicle.ID)
	assert.NotNil(t, vehicle2)
	assert.EqualValues(t, vehicle, vehicle2)

	token := GetDB().CreateAPIToken(vehicle.UserID, "pass1234")
	vehicle2 = GetDB().GetVehicleByID(vehicle.ID)
	assert.Equal(t, token, vehicle2.APIToken)

	assert.True(t, GetDB().IsTokenPasswordValid(token, "pass1234"))
	assert.False(t, GetDB().IsTokenPasswordValid(token, "pass1235"))

	GetDB().UpdateAPITokenPassword(token, "pass5678")
	assert.False(t, GetDB().IsTokenPasswordValid(token, "pass1234"))
	assert.True(t, GetDB().IsTokenPasswordValid(token, "pass5678"))

	userID := GetDB().GetAPITokenUserID(token)
	assert.Equal(t, vehicle.UserID, userID)

	GetDB().DeleteVehicle(vehicle.ID)
	vehicle2 = GetDB().GetVehicleByID(vehicle.ID)
	assert.Nil(t, vehicle2)
}

func TestDB_GetVehicles(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v1 := &Vehicle{ID: 1, UserID: "a", DisplayName: "V 1"}
	v2 := &Vehicle{ID: 2, UserID: "a", DisplayName: "V 2"}
	v3 := &Vehicle{ID: 3, UserID: "b", DisplayName: "V 3"}

	GetDB().CreateUpdateVehicle(v1)
	GetDB().CreateUpdateVehicle(v2)
	GetDB().CreateUpdateVehicle(v3)

	l1 := GetDB().GetVehicles("a")
	l2 := GetDB().GetVehicles("b")
	l3 := GetDB().GetAllVehicles()

	assert.Len(t, l1, 2)
	assert.Len(t, l2, 1)
	assert.Len(t, l3, 3)

	assert.Equal(t, v1.ID, l1[0].ID)
	assert.Equal(t, v2.ID, l1[1].ID)

	assert.Equal(t, v3.ID, l2[0].ID)

	assert.Equal(t, v1.ID, l3[0].ID)
	assert.Equal(t, v2.ID, l3[1].ID)
	assert.Equal(t, v3.ID, l3[2].ID)
}

func TestDB_CRUDVehicleState(t *testing.T) {
	t.Cleanup(ResetTestDB)
	vehicleID := 123

	state := GetDB().GetVehicleState(vehicleID)
	assert.Nil(t, state)

	GetDB().SetVehicleStatePluggedIn(vehicleID, true)
	GetDB().SetVehicleStateCharging(vehicleID, ChargeStateChargingOnGrid)
	GetDB().SetVehicleStateSoC(vehicleID, 22)
	GetDB().SetVehicleStateAmps(vehicleID, 5)

	state = GetDB().GetVehicleState(vehicleID)
	assert.NotNil(t, state)
	assert.Equal(t, vehicleID, state.VehicleID)
	assert.Equal(t, true, state.PluggedIn)
	assert.Equal(t, ChargeStateChargingOnGrid, state.Charging)
	assert.Equal(t, 22, state.SoC)
	assert.Equal(t, 5, state.Amps)

	GetDB().SetVehicleStatePluggedIn(vehicleID, false)
	GetDB().SetVehicleStateCharging(vehicleID, ChargeStateNotCharging)
	GetDB().SetVehicleStateSoC(vehicleID, 23)
	GetDB().SetVehicleStateAmps(vehicleID, 6)

	state = GetDB().GetVehicleState(vehicleID)
	assert.NotNil(t, state)
	assert.Equal(t, vehicleID, state.VehicleID)
	assert.Equal(t, false, state.PluggedIn)
	assert.Equal(t, ChargeStateNotCharging, state.Charging)
	assert.Equal(t, 23, state.SoC)
	assert.Equal(t, 6, state.Amps)
}

func TestDB_IsUserOwnerOfVehicle(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v1 := &Vehicle{ID: 1, UserID: "a", DisplayName: "V 1"}
	v2 := &Vehicle{ID: 2, UserID: "a", DisplayName: "V 2"}
	v3 := &Vehicle{ID: 3, UserID: "b", DisplayName: "V 3"}

	GetDB().CreateUpdateVehicle(v1)
	GetDB().CreateUpdateVehicle(v2)
	GetDB().CreateUpdateVehicle(v3)

	assert.True(t, GetDB().IsUserOwnerOfVehicle("a", 1))
	assert.True(t, GetDB().IsUserOwnerOfVehicle("a", 2))
	assert.True(t, GetDB().IsUserOwnerOfVehicle("b", 3))

	assert.False(t, GetDB().IsUserOwnerOfVehicle("b", 1))
	assert.False(t, GetDB().IsUserOwnerOfVehicle("b", 2))
	assert.False(t, GetDB().IsUserOwnerOfVehicle("a", 3))
}

func TestDB_GetVehicleIDsWithTibberTokenWithoutPricesForTomorrow(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v := &Vehicle{
		ID:           1,
		UserID:       "a",
		DisplayName:  "V 1",
		GridProvider: GridProviderTibber,
		TibberToken:  "123",
	}
	GetDB().CreateUpdateVehicle(v)

	now := time.Now().UTC()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for i := 0; i <= 22; i++ {
		SetTibberTestPrice(v.ID, now.Add(time.Hour*time.Duration(i)), 0.32) // 00:00
	}

	l := GetDB().GetVehicleIDsWithTibberTokenWithoutPricesForTomorrow(45)
	assert.NotNil(t, l)
	assert.Len(t, l, 1)
	assert.Equal(t, v.ID, l[0])
}

func TestDB_encrypt(t *testing.T) {
	plaintext := "this is a test"
	in := GetDB().encrypt(plaintext)
	out := GetDB().decrypt(in)
	assert.Equal(t, plaintext, out)
}

func TestDB_UpgradeRefreshTokenEncryption(t *testing.T) {
	t.Cleanup(ResetTestDB)

	userID := "1234abcd"
	GetDB().GetConnection().Exec("replace into users values(?, ?, ?, ?, ?, ?)", userID, "", "", 0.0, 0.0, 100)
	user := GetDB().GetUser(userID)

	user.TeslaRefreshToken = "xyzabc"
	user.TeslaUserID = "009988"
	GetDB().CreateUpdateUser(user)

	user = GetDB().GetUser(userID)
	assert.Equal(t, "xyzabc", user.TeslaRefreshToken)
	assert.Equal(t, "009988", user.TeslaUserID)
}
