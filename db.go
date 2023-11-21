package main

import (
	"database/sql"
	"log"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var DB_CONNECTION *sql.DB

func ConnectDB() {
	db, err := sql.Open("sqlite", GetConfig().DBFile)
	if err != nil {
		log.Panicln(err)
	}
	DB_CONNECTION = db
}

func GetDB() *sql.DB {
	return DB_CONNECTION
}

func InitDBStructure() {
	_, err := GetDB().Exec(`
create table if not exists auth_codes(id text primary key, ts text);
	`)
	if err != nil {
		log.Panicln(err)
	}
}

func CreateAuthCode() string {
	id := uuid.New().String()
	_, err := GetDB().Exec("insert into auth_codes values(?, date())", id)
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
