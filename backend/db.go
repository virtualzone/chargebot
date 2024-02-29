package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var SQLITE_DATETIME_LAYOUT string = "2006-01-02 15:04:05"

type User struct {
	ID                string  `json:"id"`
	TeslaUserID       string  `json:"tesla_user_id"`
	TeslaRefreshToken string  `json:"tesla_refresh_token"`
	APIToken          string  `json:"api_token"`
	HomeLatitude      float64 `json:"home_lat"`
	HomeLongitude     float64 `json:"home_lng"`
	HomeRadius        int     `json:"home_radius"`
}

type Vehicle struct {
	ID                  int          `json:"id"`
	UserID              string       `json:"user_id"`
	VIN                 string       `json:"vin"`
	DisplayName         string       `json:"display_name"`
	APIToken            string       `json:"api_token"`
	Enabled             bool         `json:"enabled"`
	TargetSoC           int          `json:"target_soc"`
	MaxAmps             int          `json:"max_amps"`
	NumPhases           int          `json:"num_phases"`
	SurplusCharging     bool         `json:"surplus_charging"`
	MinSurplus          int          `json:"min_surplus"`
	MinChargeTime       int          `json:"min_chargetime"`
	LowcostCharging     bool         `json:"lowcost_charging"`
	MaxPrice            int          `json:"max_price"`
	GridProvider        GridProvider `json:"gridProvider"`
	GridStrategy        GridStrategy `json:"gridStrategy"`
	DepartDays          string       `json:"departDays"`
	DepartTime          string       `json:"departTime"`
	TibberToken         string       `json:"tibber_token"`
	TelemetryEnrollDate *time.Time   `json:"telemetry_enroll_date"`
}

type APIToken struct {
	Token     string `json:"token"`
	VehicleID string `json:"vehicle_id"`
}

type SurplusRecord struct {
	Timestamp    time.Time `json:"ts"`
	SurplusWatts int       `json:"surplus_watts"`
}

type ChargeState int

const (
	ChargeStateNotCharging     ChargeState = 0
	ChargeStateChargingOnSolar ChargeState = 1
	ChargeStateChargingOnGrid  ChargeState = 2
)

type GridStrategy int

const (
	GridStrategyNoDeparturePriceLimit   GridStrategy = 1
	GridStrategyDepartureWithPriceLimit GridStrategy = 2
	GridStrategyDepartureNoPriceLimit   GridStrategy = 3
)

type GridProvider string

const (
	GridProviderTibber GridProvider = "tibber"
)

type VehicleState struct {
	VehicleID   int         `json:"vehicle_id"`
	PluggedIn   bool        `json:"pluggedIn"`
	Charging    ChargeState `json:"chargingState"`
	SoC         int         `json:"soc"`
	Amps        int         `json:"amps"`
	ChargeLimit int         `json:"chargeLimit"`
}

type ChargingEvent struct {
	Timestamp time.Time `json:"ts"`
	Event     int       `json:"event"`
	Data      string    `json:"data"`
}

const (
	LogEventChargeStart          = 1
	LogEventChargeStop           = 2
	LogEventVehiclePlugIn        = 3
	LogEventVehicleUnplug        = 4
	LogEventVehicleUpdateData    = 5
	LogEventWakeVehicle          = 6
	LogEventSetTargetSoC         = 7
	LogEventSetChargingAmps      = 8
	LogEventSetScheduledCharging = 9
)

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
drop table if exists surpluses;
drop table if exists logs;
drop table if exists vehicle_states;
drop table if exists tibber_prices;
drop table if exists grid_hourblocks;
`)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) InitDBStructure() {
	log.Println("Initializing database structure...")
	_, err := db.GetConnection().Exec(`
create table if not exists auth_codes(id text primary key, ts text);
create table if not exists users(id text primary key, tesla_refresh_token text, tesla_user_id text default '');
create table if not exists vehicles(id int primary key, user_id text, vin text, display_name text, enabled int, target_soc int, max_amps int, surplus_charging int, min_surplus int, min_chargetime int, lowcost_charging int, max_price int, tibber_token text, num_phases int default 3, grid_provider text default 'tibber', grid_strategy int default 1, depart_days text default '12345', depart_time text default '07:00');
create table if not exists api_tokens(token text primary key, user_id text, passhash text);
create table if not exists surpluses(vehicle_id int, ts text, surplus_watts int);
create table if not exists logs(vehicle_id int, ts text, event_id int, details text);
create table if not exists vehicle_states(vehicle_id int primary key, plugged_in int default 0, charging int default 0, soc int default -1, charge_amps int default 0);
create table if not exists tibber_prices(vehicle_id int not null, hourstamp int not null, price real, primary key(vehicle_id, hourstamp));
create table if not exists grid_hourblocks(vehicle_id int not null, hourstamp int not null, primary key(vehicle_id, hourstamp));
`)
	if err != nil {
		log.Panicln(err)
	}
	if _, err := db.GetConnection().Exec(`alter table vehicles add column telemetry_enroll_date string default ''`); err != nil {
		log.Println(err)
	}
	if _, err := db.GetConnection().Exec(`alter table users add column home_lat real default 0.0`); err != nil {
		log.Println(err)
	}
	if _, err := db.GetConnection().Exec(`alter table users add column home_lng real default 0.0`); err != nil {
		log.Println(err)
	}
	if _, err := db.GetConnection().Exec(`alter table users add column home_radius real default 100`); err != nil {
		log.Println(err)
	}
	if _, err := db.GetConnection().Exec(`alter table vehicle_states add column charge_limit int default 0`); err != nil {
		log.Println(err)
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
		log.Println(err)
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
	_, err := db.GetConnection().Exec("replace into users values(?, ?, ?, ?, ?, ?)", user.ID, "c:"+db.encrypt(user.TeslaRefreshToken), user.TeslaUserID, user.HomeLatitude, user.HomeLongitude, user.HomeRadius)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetUser(ID string) *User {
	e := &User{}
	err := db.GetConnection().QueryRow("select id, tesla_refresh_token, tesla_user_id, home_lat, home_lng, home_radius, ifnull(token, '') "+
		"from users "+
		"left join api_tokens on api_tokens.user_id = users.id "+
		"where id = ?",
		ID).
		Scan(&e.ID, &e.TeslaRefreshToken, &e.TeslaUserID, &e.HomeLatitude, &e.HomeLongitude, &e.HomeRadius, &e.APIToken)
	if err != nil {
		log.Println(err)
		return nil
	}
	if strings.Index(e.TeslaRefreshToken, "c:") == 0 {
		e.TeslaRefreshToken = db.decrypt(e.TeslaRefreshToken[2:])
	}
	return e
}

func (db *DB) CreateUpdateVehicle(e *Vehicle) {
	ts := ""
	if e.TelemetryEnrollDate != nil {
		ts = db.formatSqliteDatetime(*e.TelemetryEnrollDate)
	}
	_, err := db.GetConnection().Exec("replace into vehicles values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		e.ID, e.UserID, e.VIN, e.DisplayName,
		e.Enabled, e.TargetSoC, e.MaxAmps, e.SurplusCharging, e.MinSurplus, e.MinChargeTime, e.LowcostCharging, e.MaxPrice, e.TibberToken, e.NumPhases, e.GridProvider, e.GridStrategy, e.DepartDays, e.DepartTime, ts)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetVehicleByID(ID int) *Vehicle {
	e := &Vehicle{}
	var ts string
	err := db.GetConnection().QueryRow("select id, vehicles.user_id, vin, display_name, ifnull(api_tokens.token, ''), "+
		"enabled, target_soc, max_amps, num_phases, surplus_charging, min_surplus, min_chargetime, lowcost_charging, grid_provider, grid_strategy, depart_days, depart_time, max_price, tibber_token, telemetry_enroll_date "+
		"from vehicles "+
		"left join api_tokens on api_tokens.user_id = vehicles.user_id "+
		"where vehicles.id = ?",
		ID).
		Scan(&e.ID, &e.UserID, &e.VIN, &e.DisplayName, &e.APIToken, &e.Enabled, &e.TargetSoC, &e.MaxAmps, &e.NumPhases, &e.SurplusCharging, &e.MinSurplus, &e.MinChargeTime, &e.LowcostCharging, &e.GridProvider, &e.GridStrategy, &e.DepartDays, &e.DepartTime, &e.MaxPrice, &e.TibberToken, &ts)
	if err != nil {
		log.Println(err)
		return nil
	}
	if ts != "" {
		parsedDate, _ := time.Parse(SQLITE_DATETIME_LAYOUT, ts)
		e.TelemetryEnrollDate = &parsedDate
	}
	return e
}

func (db *DB) GetVehicleByVIN(vin string) *Vehicle {
	e := &Vehicle{}
	var ts string
	err := db.GetConnection().QueryRow("select id, vehicles.user_id, vin, display_name, ifnull(api_tokens.token, ''), "+
		"enabled, target_soc, max_amps, num_phases, surplus_charging, min_surplus, min_chargetime, lowcost_charging, grid_provider, grid_strategy, depart_days, depart_time, max_price, tibber_token, telemetry_enroll_date "+
		"from vehicles "+
		"left join api_tokens on api_tokens.user_id = vehicles.user_id "+
		"where vehicles.vin = ?",
		vin).
		Scan(&e.ID, &e.UserID, &e.VIN, &e.DisplayName, &e.APIToken, &e.Enabled, &e.TargetSoC, &e.MaxAmps, &e.NumPhases, &e.SurplusCharging, &e.MinSurplus, &e.MinChargeTime, &e.LowcostCharging, &e.GridProvider, &e.GridStrategy, &e.DepartDays, &e.DepartTime, &e.MaxPrice, &e.TibberToken, &ts)
	if err != nil {
		log.Println(err)
		return nil
	}
	if ts != "" {
		parsedDate, _ := time.Parse(SQLITE_DATETIME_LAYOUT, ts)
		e.TelemetryEnrollDate = &parsedDate
	}
	return e
}

func (db *DB) GetVehicles(UserID string) []*Vehicle {
	result := []*Vehicle{}
	rows, err := db.GetConnection().Query("select id, vehicles.user_id, vin, display_name, ifnull(api_tokens.token, ''), "+
		"enabled, target_soc, max_amps, num_phases, surplus_charging, min_surplus, min_chargetime, lowcost_charging, grid_provider, grid_strategy, depart_days, depart_time, max_price, tibber_token, telemetry_enroll_date "+
		"from vehicles "+
		"left join api_tokens on api_tokens.user_id = vehicles.user_id "+
		"where vehicles.user_id = ? "+
		"order by display_name",
		UserID)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var ts string
		e := &Vehicle{}
		rows.Scan(&e.ID, &e.UserID, &e.VIN, &e.DisplayName, &e.APIToken, &e.Enabled, &e.TargetSoC, &e.MaxAmps, &e.NumPhases, &e.SurplusCharging, &e.MinSurplus, &e.MinChargeTime, &e.LowcostCharging, &e.GridProvider, &e.GridStrategy, &e.DepartDays, &e.DepartTime, &e.MaxPrice, &e.TibberToken, &ts)
		if ts != "" {
			parsedDate, _ := time.Parse(SQLITE_DATETIME_LAYOUT, ts)
			e.TelemetryEnrollDate = &parsedDate
		}
		result = append(result, e)
	}
	return result
}

func (db *DB) GetAllVehicles() []*Vehicle {
	result := []*Vehicle{}
	rows, err := db.GetConnection().Query("select id, vehicles.user_id, vin, display_name, ifnull(api_tokens.token, ''), " +
		"enabled, target_soc, max_amps, num_phases, surplus_charging, min_surplus, min_chargetime, lowcost_charging, grid_provider, grid_strategy, depart_days, depart_time, max_price, tibber_token, telemetry_enroll_date " +
		"from vehicles " +
		"left join api_tokens on api_tokens.user_id = vehicles.user_id")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var ts string
		e := &Vehicle{}
		rows.Scan(&e.ID, &e.UserID, &e.VIN, &e.DisplayName, &e.APIToken, &e.Enabled, &e.TargetSoC, &e.MaxAmps, &e.NumPhases, &e.SurplusCharging, &e.MinSurplus, &e.MinChargeTime, &e.LowcostCharging, &e.GridProvider, &e.GridStrategy, &e.DepartDays, &e.DepartTime, &e.MaxPrice, &e.TibberToken, &ts)
		if ts != "" {
			parsedDate, _ := time.Parse(SQLITE_DATETIME_LAYOUT, ts)
			e.TelemetryEnrollDate = &parsedDate
		}
		result = append(result, e)
	}
	return result
}

func (db *DB) DeleteVehicle(ID int) {
	if _, err := db.GetConnection().Exec("delete from vehicles where id = ?", ID); err != nil {
		log.Panicln(err)
	}
	if _, err := db.GetConnection().Exec("delete from surpluses where vehicle_id = ?", ID); err != nil {
		log.Panicln(err)
	}
	if _, err := db.GetConnection().Exec("delete from logs where vehicle_id = ?", ID); err != nil {
		log.Panicln(err)
	}
	if _, err := db.GetConnection().Exec("delete from vehicle_states where vehicle_id = ?", ID); err != nil {
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
		log.Println(err)
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
		log.Println(err)
		return ""
	}
	return token
}

func (db *DB) GetVehicleState(vehicleID int) *VehicleState {
	e := &VehicleState{}
	err := db.GetConnection().QueryRow("select vehicle_id, plugged_in, charging, soc, charge_amps, charge_limit from vehicle_states where vehicle_id = ?",
		vehicleID).
		Scan(&e.VehicleID, &e.PluggedIn, &e.Charging, &e.SoC, &e.Amps, &e.ChargeLimit)
	if err != nil {
		log.Println(err)
		return nil
	}
	return e
}

func (db *DB) SetVehicleStatePluggedIn(vehicleID int, pluggedIn bool) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_id, plugged_in) values(?, ?) "+
		"on conflict(vehicle_id) do update set plugged_in = ?",
		vehicleID, pluggedIn, pluggedIn)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) SetVehicleStateCharging(vehicleID int, charging ChargeState) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_id, charging) values(?, ?) "+
		"on conflict(vehicle_id) do update set charging = ?",
		vehicleID, charging, charging)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) SetVehicleStateSoC(vehicleID int, soc int) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_id, soc) values(?, ?) "+
		"on conflict(vehicle_id) do update set soc = ?",
		vehicleID, soc, soc)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) SetVehicleStateAmps(vehicleID int, amps int) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_id, charge_amps) values(?, ?) "+
		"on conflict(vehicle_id) do update set charge_amps = ?",
		vehicleID, amps, amps)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) SetVehicleStateChargeLimit(vehicleID int, limit int) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_id, charge_limit) values(?, ?) "+
		"on conflict(vehicle_id) do update set charge_limit = ?",
		vehicleID, limit, limit)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) RecordSurplus(vehicleID int, surplus int) {
	_, err := db.GetConnection().Exec("insert into surpluses (vehicle_id, ts, surplus_watts) values (?, ?, ?)", vehicleID, db.formatSqliteDatetime(db.Time.UTCNow()), surplus)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetLatestSurplusRecords(vehicleID int, num int) []*SurplusRecord {
	result := []*SurplusRecord{}
	rows, err := db.GetConnection().Query("select ts, surplus_watts "+
		"from surpluses where vehicle_id = ? order by ts desc limit ?",
		vehicleID, num)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var ts string
		var surplus int
		rows.Scan(&ts, &surplus)
		parsedTime, _ := time.Parse(SQLITE_DATETIME_LAYOUT, ts)
		e := &SurplusRecord{
			Timestamp:    parsedTime,
			SurplusWatts: surplus,
		}
		result = append(result, e)
	}
	return result
}

func (db *DB) RecordSelectedGridHourblock(vehicleID int, year int, month int, day int, hour int) {
	hourstamp := GetHourstamp(year, month, day, hour)
	_, err := db.GetConnection().Exec("replace into grid_hourblocks (vehicle_id, hourstamp) values(?, ?)",
		vehicleID, hourstamp)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) IsSelectedGridHourblock(vehicleID int, year int, month int, day int, hour int) bool {
	hourstamp := GetHourstamp(year, month, day, hour)
	var num int
	err := db.GetConnection().QueryRow("select count(*) from grid_hourblocks where vehicle_id = ? and hourstamp = ?",
		vehicleID, hourstamp).Scan(&num)
	if err != nil {
		log.Fatalln(err)
		return false
	}
	return num > 0

}

func (db *DB) SetTibberPrice(vehicleID int, year int, month int, day int, hour int, price float32) {
	hourstamp := GetHourstamp(year, month, day, hour)
	_, err := db.GetConnection().Exec("replace into tibber_prices (vehicle_id, hourstamp, price) values(?, ?, ?)",
		vehicleID, hourstamp, price)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) GetUpcomingTibberPrices(vehicleID int, sortByPriceAsc bool) []*GridPrice {
	now := db.Time.UTCNow()
	hourstampStart := GetHourstamp(now.Year(), int(now.Month()), now.Day(), now.Hour())
	result := []*GridPrice{}
	order := "hourstamp asc"
	if sortByPriceAsc {
		order = "price asc"
	}
	rows, err := db.GetConnection().Query("select hourstamp, price "+
		"from tibber_prices "+
		"where vehicle_id = ? and hourstamp >= ?"+
		"order by "+order,
		vehicleID, hourstampStart)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var hourstamp string
		var price float32
		rows.Scan(&hourstamp, &price)

		year, _ := strconv.Atoi(hourstamp[0:4])
		month, _ := strconv.Atoi(hourstamp[4:6])
		day, _ := strconv.Atoi(hourstamp[6:8])
		hour, _ := strconv.Atoi(hourstamp[8:])

		ts := time.Date(year, time.Month(month), day, hour, 0, 0, 0, now.Location())

		e := &GridPrice{
			Total:    price,
			StartsAt: ts,
		}
		result = append(result, e)
	}
	return result
}

func (db *DB) GetVehicleIDsWithTibberTokenWithoutPricesForStarttime(startTime time.Time, limit int) []int {
	hourstampStart := GetHourstamp(startTime.Year(), int(startTime.Month()), startTime.Day(), 0)
	result := []int{}
	rows, err := db.GetConnection().Query("select vehicles.id "+
		"from vehicles "+
		"where vehicles.grid_provider = 'tibber' and ifnull(vehicles.tibber_token, '') != '' and (select count(*) from tibber_prices where tibber_prices.vehicle_id = vehicles.id and hourstamp >= ?) = 0 "+
		"limit ?",
		hourstampStart, limit)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var vehicleID int
		rows.Scan(&vehicleID)
		result = append(result, vehicleID)
	}
	return result
}

func (db *DB) GetVehicleIDsWithTibberTokenWithoutPricesForTomorrow(limit int) []int {
	startTime := db.Time.UTCNow().AddDate(0, 0, 1)
	return db.GetVehicleIDsWithTibberTokenWithoutPricesForStarttime(startTime, limit)
}

func (db *DB) GetVehicleIDsWithTibberTokenWithoutPricesForToday(limit int) []int {
	startTime := db.Time.UTCNow()
	return db.GetVehicleIDsWithTibberTokenWithoutPricesForStarttime(startTime, limit)
}

func (db *DB) IsUserOwnerOfVehicle(userID string, vehicleID int) bool {
	list := db.GetVehicles(userID)
	for _, e := range list {
		if e.ID == vehicleID {
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
		log.Println(err)
		return false
	}
	return IsValidHash(password, passhash)
}

func (db *DB) LogChargingEvent(vehicleID int, eventType int, text string) {
	log.Printf("charging event %d for vehicle id %d with data: %s\n", eventType, vehicleID, text)
	_, err := db.GetConnection().Exec("insert into logs values(?, ?, ?, ?)", vehicleID, db.formatSqliteDatetime(db.Time.UTCNow()), eventType, text)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetLatestChargingEvent(vehicleID int, eventType int) *ChargingEvent {
	var ts string
	var eventId int
	var details string
	err := db.GetConnection().QueryRow("select ts, event_id, details "+
		"from logs where vehicle_id = ? and event_id = ? order by ts desc limit 1",
		vehicleID, eventType).
		Scan(&ts, &eventId, &details)
	if err != nil {
		log.Println(err)
		return nil
	}
	parsedTime, _ := time.Parse(SQLITE_DATETIME_LAYOUT, ts)
	e := &ChargingEvent{
		Timestamp: parsedTime,
		Event:     eventId,
		Data:      details,
	}
	return e
}

func (db *DB) GetLatestChargingEvents(vehicleID int, num int) []*ChargingEvent {
	result := []*ChargingEvent{}
	rows, err := db.GetConnection().Query("select ts, event_id, details "+
		"from logs where vehicle_id = ? order by ts desc limit ?",
		vehicleID, num)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var ts string
		var eventId int
		var details string
		rows.Scan(&ts, &eventId, &details)
		parsedTime, _ := time.Parse(SQLITE_DATETIME_LAYOUT, ts)
		e := &ChargingEvent{
			Timestamp: parsedTime,
			Event:     eventId,
			Data:      details,
		}
		result = append(result, e)
	}
	return result
}

func (db *DB) formatSqliteDatetime(ts time.Time) string {
	return ts.Format(SQLITE_DATETIME_LAYOUT)
}

/*
func (db *DB) parseSqliteDatetime(ts string) *time.Time {
	parsedTime, _ := time.Parse(SQLITE_DATETIME_LAYOUT, ts)
	return &parsedTime
}
*/

func (db *DB) encrypt(s string) string {
	aes, err := aes.NewCipher([]byte(GetConfig().CryptKey))
	if err != nil {
		panic(err)
	}
	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		panic(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(s), nil)
	res := base64.StdEncoding.EncodeToString(ciphertext)
	return res
}

func (db *DB) decrypt(s string) string {
	ciphertext, _ := base64.StdEncoding.Strict().DecodeString(s)
	aes, err := aes.NewCipher([]byte(GetConfig().CryptKey))
	if err != nil {
		panic(err)
	}
	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		panic(err)
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	if err != nil {
		panic(err)
	}

	return string(plaintext)
}
