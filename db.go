package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"math/rand"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type User struct {
	ID           string `json:"id"`
	RefreshToken string `json:"refresh_token"`
}

type Vehicle struct {
	ID              int    `json:"id"`
	UserID          string `json:"user_id"`
	VIN             string `json:"vin"`
	DisplayName     string `json:"display_name"`
	APIToken        string `json:"api_token"`
	Enabled         bool   `json:"enabled"`
	TargetSoC       int    `json:"target_soc"`
	SurplusCharging bool   `json:"surplus_charging"`
	MinSurplus      int    `json:"min_surplus"`
	MinChargeTime   int    `json:"min_chargetime"`
	LowcostCharging bool   `json:"lowcost_charging"`
	MaxPrice        int    `json:"max_price"`
}

type APIToken struct {
	Token     string `json:"token"`
	VehicleID string `json:"vehicle_id"`
}

var DB_CONNECTION *sql.DB

func ConnectDB() {
	db, err := sql.Open("sqlite", GetConfig().DBFile+"?_pragma=busy_timeout=10000&_pragma=journal_mode=WAL")
	if err != nil {
		log.Panicln(err)
	}
	db.SetMaxOpenConns(10000)
	db.SetMaxIdleConns(10000)
	DB_CONNECTION = db
}

func GetDB() *sql.DB {
	return DB_CONNECTION
}

func InitDBStructure() {
	_, err := GetDB().Exec(`
create table if not exists auth_codes(id text primary key, ts text);
create table if not exists users(id text primary key, refresh_token text);
create table if not exists vehicles(id int primary key, user_id text, vin text, display_name text, enabled int, target_soc int, surplus_charging int, min_surplus int, min_chargetime int, lowcost_charging int, max_price int);
create table if not exists api_tokens(token text primary key, vehicle_id int, passhash text);
create table if not exists surpluses(vehicle_id int, ts text, surplus_watts int);
`)
	if err != nil {
		log.Panicln(err)
	}
}

func CreateAuthCode() string {
	id := uuid.New().String()
	_, err := GetDB().Exec("insert into auth_codes values(?, datetime())", id)
	if err != nil {
		log.Panicln(err)
	}
	return id
}

func IsValidAuthCode(code string) bool {
	row := GetDB().QueryRow("select count(*) from auth_codes where id = ?", code)
	var count int
	if err := row.Scan(&count); err != nil {
		log.Println(err)
		return false
	}
	return count == 1
}

func DeleteAuthCode(code string) {
	_, err := GetDB().Exec("delete from auth_codes where id = ?", code)
	if err != nil {
		log.Panicln(err)
	}
}

func DeleteExpiredAuthCodes() {
	_, err := GetDB().Exec("delete from auth_codes where ts < date('now', '-15 minutes')")
	if err != nil {
		log.Panicln(err)
	}
}

func CreateUpdateUser(user *User) {
	_, err := GetDB().Exec("replace into users values(?, ?)", user.ID, user.RefreshToken)
	if err != nil {
		log.Panicln(err)
	}
}

func GetUser(ID string) *User {
	e := &User{}
	err := GetDB().QueryRow("select id, refresh_token from users where id = ?",
		ID).
		Scan(&e.ID, &e.RefreshToken)
	if err != nil {
		log.Println(err)
		return nil
	}
	return e
}

func CreateUpdateVehicle(e *Vehicle) {
	_, err := GetDB().Exec("replace into vehicles values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		e.ID, e.UserID, e.VIN, e.DisplayName,
		e.Enabled, e.TargetSoC, e.SurplusCharging, e.MinSurplus, e.MinChargeTime, e.LowcostCharging, e.MaxPrice)
	if err != nil {
		log.Panicln(err)
	}
}

func GetVehicleByID(ID int) *Vehicle {
	e := &Vehicle{}
	err := GetDB().QueryRow("select id, user_id, vin, display_name, api_tokens.token, "+
		"enabled, target_soc, surplus_charging, min_surplus, min_chargetime, lowcost_charging, max_price "+
		"from vehicles "+
		"left join api_tokens on api_tokens.vehicle_id = vehicles.id "+
		"where vehicles.id = ?",
		ID).
		Scan(&e.ID, &e.UserID, &e.VIN, &e.DisplayName, &e.APIToken, &e.Enabled, &e.TargetSoC, &e.SurplusCharging, &e.MinSurplus, &e.MinChargeTime, &e.LowcostCharging, &e.MaxPrice)
	if err != nil {
		log.Println(err)
		return nil
	}
	return e
}

func GetVehicles(UserID string) []*Vehicle {
	result := []*Vehicle{}
	rows, err := GetDB().Query("select id, user_id, vin, display_name, api_tokens.token, "+
		"enabled, target_soc, surplus_charging, min_surplus, min_chargetime, lowcost_charging, max_price "+
		"from vehicles "+
		"left join api_tokens on api_tokens.vehicle_id = vehicles.id "+
		"where user_id = ?",
		UserID)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		e := &Vehicle{}
		rows.Scan(&e.ID, &e.UserID, &e.VIN, &e.DisplayName, &e.APIToken, &e.Enabled, &e.TargetSoC, &e.SurplusCharging, &e.MinSurplus, &e.MinChargeTime, &e.LowcostCharging, &e.MaxPrice)
		result = append(result, e)
	}
	return result
}

func DeleteVehicle(ID int) {
	_, err := GetDB().Exec("delete from vehicles where id = ?", ID)
	if err != nil {
		log.Panicln(err)
	}
}

func CreateAPIToken(VehicleID int, password string) string {
	id := uuid.New().String()
	passhash := GetSHA256Hash(password)
	_, err := GetDB().Exec("insert into api_tokens values(?, ?, ?)", id, VehicleID, passhash)
	if err != nil {
		log.Panicln(err)
	}
	return id
}

func UpdateAPITokenPassword(token string, password string) {
	passhash := GetSHA256Hash(password)
	_, err := GetDB().Exec("update api_tokens set passhash = ? where token = ?", passhash, token)
	if err != nil {
		log.Panicln(err)
	}
}

func GetAPITokenVehicleID(token string) int {
	var vehicleID int
	err := GetDB().QueryRow("select vehicle_id from api_tokens where token = ?",
		token).
		Scan(&vehicleID)
	if err != nil {
		log.Println(err)
		return 0
	}
	return vehicleID
}

func RecordSurplus(vehicleID int, surplus int) {
	_, err := GetDB().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, datetime(), ?)", vehicleID, surplus)
	if err != nil {
		log.Panicln(err)
	}
}

func IsUserOwnerOfVehicle(userID string, vehicleID int) bool {
	list := GetVehicles(userID)
	for _, e := range list {
		if e.UserID == userID {
			return true
		}
	}
	return false
}

func IsTokenPasswordValid(token string, password string) bool {
	var passhash string
	err := GetDB().QueryRow("select passhash "+
		"from api_tokens "+
		"where token = ?",
		token).Scan(&passhash)
	if err != nil {
		log.Println(err)
		return false
	}
	return IsValidHash(password, passhash)
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

func GetSHA256Hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func IsValidHash(plain string, hash string) bool {
	s := GetSHA256Hash(plain)
	return s == hash
}
