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

	_ "modernc.org/sqlite"
)

var SQLITE_DATETIME_LAYOUT string = "2006-01-02 15:04:05"

type Vehicle struct {
	VIN                 string       `json:"vin"`
	DisplayName         string       `json:"display_name"`
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
	VIN         string      `json:"vehicle_vin"`
	PluggedIn   bool        `json:"pluggedIn"`
	Charging    ChargeState `json:"chargingState"`
	SoC         int         `json:"soc"`
	Amps        int         `json:"amps"`
	ChargeLimit int         `json:"chargeLimit"`
	IsHome      bool        `json:"is_home"`
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

const (
	SettingRefreshToken = "refresh_token"
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
drop table if exists settings;
drop table if exists vehicles;
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
create table if not exists settings(key text primary key, value text default '');
create table if not exists vehicles(vin text primary key, display_name text, enabled int, target_soc int, max_amps int, surplus_charging int, min_surplus int, min_chargetime int, lowcost_charging int, max_price int, tibber_token text, num_phases int default 3, grid_provider text default 'tibber', grid_strategy int default 1, depart_days text default '12345', depart_time text default '07:00', telemetry_enroll_date string default '');
create table if not exists surpluses(ts text, surplus_watts int);
create table if not exists logs(vehicle_vin text, ts text, event_id int, details text);
create table if not exists vehicle_states(vehicle_vin text primary key, plugged_in int default 0, charging int default 0, soc int default -1, charge_amps int default 0, charge_limit int default 0, is_home int default 0);
create table if not exists tibber_prices(vehicle_vin text not null, hourstamp int not null, price real, primary key(vehicle_vin, hourstamp));
create table if not exists grid_hourblocks(vehicle_vin int text null, hourstamp int not null, primary key(vehicle_vin, hourstamp));
`)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) SetSetting(key, value string) {
	if key == SettingRefreshToken && GetConfig().CryptKey != "" {
		value = "c:" + db.encrypt(value)
	}
	_, err := db.GetConnection().Exec("replace into settings values(?, ?)",
		key, value)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetSetting(key string) string {
	var value string
	err := db.GetConnection().QueryRow("select value from settings where key = ?", key).
		Scan(&value)
	if err != nil {
		log.Println(err)
		return ""
	}
	if key == SettingRefreshToken && GetConfig().CryptKey != "" && strings.Index(value, "c:") == 0 {
		value = db.decrypt(value[2:])
	}
	return value
}

func (db *DB) CreateUpdateVehicle(e *Vehicle) {
	ts := ""
	if e.TelemetryEnrollDate != nil {
		ts = db.formatSqliteDatetime(*e.TelemetryEnrollDate)
	}
	_, err := db.GetConnection().Exec("replace into vehicles values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		e.VIN, e.DisplayName,
		e.Enabled, e.TargetSoC, e.MaxAmps, e.SurplusCharging, e.MinSurplus, e.MinChargeTime, e.LowcostCharging, e.MaxPrice, e.TibberToken, e.NumPhases, e.GridProvider, e.GridStrategy, e.DepartDays, e.DepartTime, ts)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetVehicleByVIN(vin string) *Vehicle {
	e := &Vehicle{}
	var ts string
	err := db.GetConnection().QueryRow("select vin, display_name, "+
		"enabled, target_soc, max_amps, num_phases, surplus_charging, min_surplus, min_chargetime, lowcost_charging, grid_provider, grid_strategy, depart_days, depart_time, max_price, tibber_token, telemetry_enroll_date "+
		"from vehicles "+
		"where vehicles.vin = ?",
		vin).
		Scan(&e.VIN, &e.DisplayName, &e.Enabled, &e.TargetSoC, &e.MaxAmps, &e.NumPhases, &e.SurplusCharging, &e.MinSurplus, &e.MinChargeTime, &e.LowcostCharging, &e.GridProvider, &e.GridStrategy, &e.DepartDays, &e.DepartTime, &e.MaxPrice, &e.TibberToken, &ts)
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

func (db *DB) GetVehicles() []*Vehicle {
	result := []*Vehicle{}
	rows, err := db.GetConnection().Query("select vin, display_name, " +
		"enabled, target_soc, max_amps, num_phases, surplus_charging, min_surplus, min_chargetime, lowcost_charging, grid_provider, grid_strategy, depart_days, depart_time, max_price, tibber_token, telemetry_enroll_date " +
		"from vehicles " +
		"order by display_name")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var ts string
		e := &Vehicle{}
		rows.Scan(&e.VIN, &e.DisplayName, &e.Enabled, &e.TargetSoC, &e.MaxAmps, &e.NumPhases, &e.SurplusCharging, &e.MinSurplus, &e.MinChargeTime, &e.LowcostCharging, &e.GridProvider, &e.GridStrategy, &e.DepartDays, &e.DepartTime, &e.MaxPrice, &e.TibberToken, &ts)
		if ts != "" {
			parsedDate, _ := time.Parse(SQLITE_DATETIME_LAYOUT, ts)
			e.TelemetryEnrollDate = &parsedDate
		}
		result = append(result, e)
	}
	return result
}

func (db *DB) DeleteVehicle(vin string) {
	if _, err := db.GetConnection().Exec("delete from vehicles where vin = ?", vin); err != nil {
		log.Panicln(err)
	}
	if _, err := db.GetConnection().Exec("delete from logs where vehicle_vin = ?", vin); err != nil {
		log.Panicln(err)
	}
	if _, err := db.GetConnection().Exec("delete from vehicle_states where vehicle_vin = ?", vin); err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetVehicleState(vin string) *VehicleState {
	e := &VehicleState{}
	err := db.GetConnection().QueryRow("select vehicle_vin, plugged_in, charging, soc, charge_amps, charge_limit, is_home from vehicle_states where vehicle_vin = ?",
		vin).
		Scan(&e.VIN, &e.PluggedIn, &e.Charging, &e.SoC, &e.Amps, &e.ChargeLimit, &e.IsHome)
	if err != nil {
		log.Println(err)
		return nil
	}
	return e
}

func (db *DB) SetVehicleStatePluggedIn(vin string, pluggedIn bool) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_vin, plugged_in) values(?, ?) "+
		"on conflict(vehicle_vin) do update set plugged_in = ?",
		vin, pluggedIn, pluggedIn)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) SetVehicleStateCharging(vin string, charging ChargeState) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_vin, charging) values(?, ?) "+
		"on conflict(vehicle_vin) do update set charging = ?",
		vin, charging, charging)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) SetVehicleStateSoC(vin string, soc int) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_vin, soc) values(?, ?) "+
		"on conflict(vehicle_vin) do update set soc = ?",
		vin, soc, soc)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) SetVehicleStateAmps(vin string, amps int) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_vin, charge_amps) values(?, ?) "+
		"on conflict(vehicle_vin) do update set charge_amps = ?",
		vin, amps, amps)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) SetVehicleStateChargeLimit(vin string, limit int) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_vin, charge_limit) values(?, ?) "+
		"on conflict(vehicle_vin) do update set charge_limit = ?",
		vin, limit, limit)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) SetVehicleStateIsHome(vin string, isHome bool) {
	_, err := db.GetConnection().Exec("insert into vehicle_states (vehicle_vin, is_home) values(?, ?) "+
		"on conflict(vehicle_vin) do update set is_home = ?",
		vin, isHome, isHome)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) RecordSurplus(surplus int) {
	_, err := db.GetConnection().Exec("insert into surpluses (ts, surplus_watts) values (?, ?)", db.formatSqliteDatetime(db.Time.UTCNow()), surplus)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetLatestSurplusRecords(num int) []*SurplusRecord {
	result := []*SurplusRecord{}
	rows, err := db.GetConnection().Query("select ts, surplus_watts "+
		"from surpluses order by ts desc limit ?",
		num)
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

func (db *DB) RecordSelectedGridHourblock(vin string, year int, month int, day int, hour int) {
	hourstamp := GetHourstamp(year, month, day, hour)
	_, err := db.GetConnection().Exec("replace into grid_hourblocks (vehicle_vin, hourstamp) values(?, ?)",
		vin, hourstamp)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) IsSelectedGridHourblock(vin string, year int, month int, day int, hour int) bool {
	hourstamp := GetHourstamp(year, month, day, hour)
	var num int
	err := db.GetConnection().QueryRow("select count(*) from grid_hourblocks where vehicle_vin = ? and hourstamp = ?",
		vin, hourstamp).Scan(&num)
	if err != nil {
		log.Fatalln(err)
		return false
	}
	return num > 0

}

func (db *DB) SetTibberPrice(vin string, year int, month int, day int, hour int, price float32) {
	hourstamp := GetHourstamp(year, month, day, hour)
	_, err := db.GetConnection().Exec("replace into tibber_prices (vehicle_vin, hourstamp, price) values(?, ?, ?)",
		vin, hourstamp, price)
	if err != nil {
		log.Fatalln(err)
	}
}

func (db *DB) GetUpcomingTibberPrices(vin string, sortByPriceAsc bool) []*GridPrice {
	now := db.Time.UTCNow()
	hourstampStart := GetHourstamp(now.Year(), int(now.Month()), now.Day(), now.Hour())
	result := []*GridPrice{}
	order := "hourstamp asc"
	if sortByPriceAsc {
		order = "price asc"
	}
	rows, err := db.GetConnection().Query("select hourstamp, price "+
		"from tibber_prices "+
		"where vehicle_vin = ? and hourstamp >= ?"+
		"order by "+order,
		vin, hourstampStart)
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

func (db *DB) GetVehicleVINsWithTibberTokenWithoutPricesForStarttime(startTime time.Time, limit int) []string {
	hourstampStart := GetHourstamp(startTime.Year(), int(startTime.Month()), startTime.Day(), 0)
	result := []string{}
	rows, err := db.GetConnection().Query("select vehicles.vin "+
		"from vehicles "+
		"where vehicles.grid_provider = 'tibber' and ifnull(vehicles.tibber_token, '') != '' and (select count(*) from tibber_prices where tibber_prices.vehicle_vin = vehicles.vin and hourstamp >= ?) = 0 "+
		"limit ?",
		hourstampStart, limit)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var vin string
		rows.Scan(&vin)
		result = append(result, vin)
	}
	return result
}

func (db *DB) GetVehicleVINsWithTibberTokenWithoutPricesForTomorrow(limit int) []string {
	startTime := db.Time.UTCNow().AddDate(0, 0, 1)
	return db.GetVehicleVINsWithTibberTokenWithoutPricesForStarttime(startTime, limit)
}

func (db *DB) GetVehicleVINsWithTibberTokenWithoutPricesForToday(limit int) []string {
	startTime := db.Time.UTCNow()
	return db.GetVehicleVINsWithTibberTokenWithoutPricesForStarttime(startTime, limit)
}

func (db *DB) LogChargingEvent(vin string, eventType int, text string) {
	log.Printf("charging event %d for vehicle id %s with data: %s\n", eventType, vin, text)
	_, err := db.GetConnection().Exec("insert into logs values(?, ?, ?, ?)", vin, db.formatSqliteDatetime(db.Time.UTCNow()), eventType, text)
	if err != nil {
		log.Panicln(err)
	}
}

func (db *DB) GetLatestChargingEvent(vin string, eventType int) *ChargingEvent {
	var ts string
	var eventId int
	var details string
	err := db.GetConnection().QueryRow("select ts, event_id, details "+
		"from logs where vehicle_vin = ? and event_id = ? order by ts desc limit 1",
		vin, eventType).
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

func (db *DB) GetLatestChargingEvents(vin string, num int) []*ChargingEvent {
	result := []*ChargingEvent{}
	rows, err := db.GetConnection().Query("select ts, event_id, details "+
		"from logs where vehicle_vin = ? order by ts desc limit ?",
		vin, num)
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
