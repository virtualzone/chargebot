package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	. "github.com/virtualzone/chargebot/goshared"
)

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
		VIN:      "789",
		UserID:   "456",
		APIToken: "",
	}
	GetDB().CreateUpdateVehicle(vehicle)

	vehicle2 := GetDB().GetVehicleByVIN(vehicle.VIN)
	assert.NotNil(t, vehicle2)
	assert.EqualValues(t, vehicle, vehicle2)

	token := GetDB().CreateAPIToken(vehicle.UserID, "pass1234")
	vehicle2 = GetDB().GetVehicleByVIN(vehicle.VIN)
	assert.Equal(t, token, vehicle2.APIToken)

	assert.True(t, GetDB().IsTokenPasswordValid(token, "pass1234"))
	assert.False(t, GetDB().IsTokenPasswordValid(token, "pass1235"))

	GetDB().UpdateAPITokenPassword(token, "pass5678")
	assert.False(t, GetDB().IsTokenPasswordValid(token, "pass1234"))
	assert.True(t, GetDB().IsTokenPasswordValid(token, "pass5678"))

	userID := GetDB().GetAPITokenUserID(token)
	assert.Equal(t, vehicle.UserID, userID)

	GetDB().DeleteVehicle(vehicle.VIN)
	vehicle2 = GetDB().GetVehicleByVIN(vehicle.VIN)
	assert.Nil(t, vehicle2)
}

func TestDB_GetVehicles(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v1 := &Vehicle{VIN: "1", UserID: "a"}
	v2 := &Vehicle{VIN: "2", UserID: "a"}
	v3 := &Vehicle{VIN: "3", UserID: "b"}

	GetDB().CreateUpdateVehicle(v1)
	GetDB().CreateUpdateVehicle(v2)
	GetDB().CreateUpdateVehicle(v3)

	l1 := GetDB().GetVehicles("a")
	l2 := GetDB().GetVehicles("b")
	l3 := GetDB().GetAllVehicles()

	assert.Len(t, l1, 2)
	assert.Len(t, l2, 1)
	assert.Len(t, l3, 3)

	assert.Equal(t, v1.VIN, l1[0].VIN)
	assert.Equal(t, v2.VIN, l1[1].VIN)

	assert.Equal(t, v3.VIN, l2[0].VIN)

	assert.Equal(t, v1.VIN, l3[0].VIN)
	assert.Equal(t, v2.VIN, l3[1].VIN)
	assert.Equal(t, v3.VIN, l3[2].VIN)
}

func TestDB_IsUserOwnerOfVehicle(t *testing.T) {
	t.Cleanup(ResetTestDB)

	v1 := &Vehicle{VIN: "1", UserID: "a"}
	v2 := &Vehicle{VIN: "2", UserID: "a"}
	v3 := &Vehicle{VIN: "3", UserID: "b"}

	GetDB().CreateUpdateVehicle(v1)
	GetDB().CreateUpdateVehicle(v2)
	GetDB().CreateUpdateVehicle(v3)

	assert.True(t, GetDB().IsUserOwnerOfVehicle("a", "1"))
	assert.True(t, GetDB().IsUserOwnerOfVehicle("a", "2"))
	assert.True(t, GetDB().IsUserOwnerOfVehicle("b", "3"))

	assert.False(t, GetDB().IsUserOwnerOfVehicle("b", "1"))
	assert.False(t, GetDB().IsUserOwnerOfVehicle("b", "2"))
	assert.False(t, GetDB().IsUserOwnerOfVehicle("a", "3"))
}

func TestDB_CRUDTelemetryState(t *testing.T) {
	t.Cleanup(ResetTestDB)

	s := &PersistedTelemetryState{
		VIN:         "abc",
		PluggedIn:   true,
		Charging:    true,
		SoC:         63,
		Amps:        10,
		ChargeLimit: 80,
		IsHome:      true,
		UTC:         time.Now().UTC().Unix(),
	}
	GetDB().SaveTelemetryState(s)

	s2 := GetDB().GetTelemetryState(s.VIN)
	assert.Equal(t, s, s2)
}
