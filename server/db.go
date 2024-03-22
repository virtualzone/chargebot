package main

import (
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	. "github.com/virtualzone/chargebot/goshared"
	_ "modernc.org/sqlite"
)

var SQLITE_DATETIME_LAYOUT string = "2006-01-02 15:04:05"

type User struct {
	ID            string  `json:"id"`
	TeslaUserID   string  `json:"tesla_user_id"`
	APIToken      string  `json:"api_token"`
	HomeLatitude  float64 `json:"home_lat"`
	HomeLongitude float64 `json:"home_lng"`
	HomeRadius    int     `json:"home_radius"`
}

type Vehicle struct {
	VIN      string `json:"vin"`
	UserID   string `json:"user_id"`
	APIToken string `json:"api_token"`
}

type DB struct {
	Connection *sql.DB
	Time       Time
}

var _DBInstance *DB
var _DBOnce sync.Once

func GetDB() *DB {
	_DBOnce.Do(func() {
		_DBInstance = &DB{
			Time: new(RealTime),
		}
	})
	return _DBInstance
}

func (db *DB) Connect() {
	log.Println("Connecting to database...")
	con, err := sql.Open("sqlite", GetConfig().DBFile+"?_pragma=busy_timeout=10000&_pragma=journal_mode=WAL")
	if err != nil {
		log.Panicln(err)
	}
	con.SetMaxOpenConns(10000)
	con.SetMaxIdleConns(10000)
	db.Connection = con
}

func (db *DB) GetConnection() *sql.DB {
	return db.Connection
}

func (db *DB) ResetDBStructure() {
	log.Println("Resetting database...")
	_, err := db.GetConnection().Exec(`
drop table if exists auth_codes;
drop table if exists users;
drop table if exists vehicles;
drop table if exists api_tokens;
drop table if exists telemetry_state;
`)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) InitDBStructure() {
	log.Println("Initializing database structure...")
	_, err := db.GetConnection().Exec(`
create table if not exists auth_codes(id text primary key, ts text);
create table if not exists users(id text primary key, tesla_user_id text default '', home_lat real default 0.0, home_lng real default 0.0, home_radius real default 100);
create table if not exists vehicles(vin text primary key, user_id text);
create table if not exists api_tokens(token text primary key, user_id text, passhash text);
create table if not exists telemetry_state(vin text primary key, plugged_in int, charging int, soc int, amps int, charge_limit int, is_home int, ts int);
`)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) CreateAuthCode() string {
	id := uuid.New().String()
	_, err := db.GetConnection().Exec("insert into auth_codes values(?, ?)", id, db.formatSqliteDatetime(db.Time.UTCNow()))
	if err != nil {
		log.Panicln(err)
	}
	return id
}

func (db *DB) IsValidAuthCode(code string) bool {
	row := db.GetConnection().QueryRow("select count(*) from auth_codes where id = ?", code)
	var count int
	if err := row.Scan(&count); err != nil {
		if err != sql.ErrNoRows {
			log.Println(err)
		}
		return false
	}
	return count == 1
}

func (db *DB) DeleteAuthCode(code string) {
	_, err := db.GetConnection().Exec("delete from auth_codes where id = ?", code)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) DeleteExpiredAuthCodes() {
	_, err := db.GetConnection().Exec("delete from auth_codes where ts < date('now', '-15 minutes')")
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) CreateUpdateUser(user *User) {
	_, err := db.GetConnection().Exec("replace into users values(?, ?, ?, ?, ?)", user.ID, user.TeslaUserID, user.HomeLatitude, user.HomeLongitude, user.HomeRadius)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetUser(ID string) *User {
	e := &User{}
	err := db.GetConnection().QueryRow("select id, tesla_user_id, home_lat, home_lng, home_radius, ifnull(token, '') "+
		"from users "+
		"left join api_tokens on api_tokens.user_id = users.id "+
		"where id = ?",
		ID).
		Scan(&e.ID, &e.TeslaUserID, &e.HomeLatitude, &e.HomeLongitude, &e.HomeRadius, &e.APIToken)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println(err)
		}
		return nil
	}
	return e
}

func (db *DB) CreateUpdateVehicle(e *Vehicle) {
	_, err := db.GetConnection().Exec("replace into vehicles values(?, ?)",
		e.VIN, e.UserID)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetVehicleByVIN(vin string) *Vehicle {
	e := &Vehicle{}
	err := db.GetConnection().QueryRow("select vin, vehicles.user_id, ifnull(api_tokens.token, '') "+
		"from vehicles "+
		"left join api_tokens on api_tokens.user_id = vehicles.user_id "+
		"where vehicles.vin = ?",
		vin).
		Scan(&e.VIN, &e.UserID, &e.APIToken)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println(err)
		}
		return nil
	}
	return e
}

func (db *DB) GetVehicles(UserID string) []*Vehicle {
	result := []*Vehicle{}
	rows, err := db.GetConnection().Query("select vin, vehicles.user_id, ifnull(api_tokens.token, '') "+
		"from vehicles "+
		"left join api_tokens on api_tokens.user_id = vehicles.user_id "+
		"where vehicles.user_id = ? "+
		"order by vin",
		UserID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println(err)
		}
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		e := &Vehicle{}
		rows.Scan(&e.VIN, &e.UserID, &e.APIToken)
		result = append(result, e)
	}
	return result
}

func (db *DB) GetAllVehicles() []*Vehicle {
	result := []*Vehicle{}
	rows, err := db.GetConnection().Query("select vin, vehicles.user_id, ifnull(api_tokens.token, '') " +
		"from vehicles " +
		"left join api_tokens on api_tokens.user_id = vehicles.user_id")
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println(err)
		}
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		e := &Vehicle{}
		rows.Scan(&e.VIN, &e.UserID, &e.APIToken)
		result = append(result, e)
	}
	return result
}

func (db *DB) DeleteVehicle(vin string) {
	if _, err := db.GetConnection().Exec("delete from vehicles where vin = ?", vin); err != nil {
		log.Panicln(err)
	}
}

func (db *DB) CreateAPIToken(userID string, password string) string {
	id := uuid.New().String()
	passhash := GetSHA256Hash(password)
	_, err := db.GetConnection().Exec("insert into api_tokens values(?, ?, ?)", id, userID, passhash)
	if err != nil {
		log.Panicln(err)
	}
	return id
}

func (db *DB) UpdateAPITokenPassword(token string, password string) {
	passhash := GetSHA256Hash(password)
	_, err := db.GetConnection().Exec("update api_tokens set passhash = ? where token = ?", passhash, token)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetAPITokenUserID(token string) string {
	var userID string
	err := db.GetConnection().QueryRow("select user_id from api_tokens where token = ?",
		token).
		Scan(&userID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println(err)
		}
		return ""
	}
	return userID
}

func (db *DB) GetAPIToken(userID string) string {
	var token string
	err := db.GetConnection().QueryRow("select token from api_tokens where user_id = ?",
		userID).
		Scan(&token)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println(err)
		}
		return ""
	}
	return token
}

func (db *DB) IsUserOwnerOfVehicle(userID string, vin string) bool {
	list := db.GetVehicles(userID)
	for _, e := range list {
		if e.VIN == vin {
			return true
		}
	}
	return false
}

func (db *DB) IsTokenPasswordValid(token string, password string) bool {
	var passhash string
	err := db.GetConnection().QueryRow("select passhash "+
		"from api_tokens "+
		"where token = ?",
		token).Scan(&passhash)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println(err)
		}
		return false
	}
	return IsValidHash(password, passhash)
}

func (db *DB) LogChargingEvent(vin string, eventType int, text string) {
	log.Printf("charging event %d for vehicle id %s with data: %s\n", eventType, vin, text)
	_, err := db.GetConnection().Exec("insert into logs values(?, ?, ?, ?)", vin, db.formatSqliteDatetime(db.Time.UTCNow()), eventType, text)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) formatSqliteDatetime(ts time.Time) string {
	return ts.Format(SQLITE_DATETIME_LAYOUT)
}

func (db *DB) SaveTelemetryState(state *PersistedTelemetryState) {
	_, err := db.GetConnection().Exec("replace into telemetry_state values(?, ?, ?, ?, ?, ?, ?, ?)",
		state.VIN, state.PluggedIn, state.Charging, state.SoC, state.Amps, state.ChargeLimit, state.IsHome, state.UTC)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetTelemetryState(vin string) *PersistedTelemetryState {
	e := &PersistedTelemetryState{}
	err := db.GetConnection().QueryRow("select vin, plugged_in, charging, soc, amps, charge_limit, is_home, ts "+
		"from telemetry_state "+
		"where vin = ?",
		vin).Scan(&e.VIN, &e.PluggedIn, &e.Charging, &e.SoC, &e.Amps, &e.ChargeLimit, &e.IsHome, &e.UTC)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Println(err)
		}
		return nil
	}
	return e
}
