package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("DB_FILE", ":memory:")
	GetConfig().ReadConfig()
	ConnectDB()
	InitDBStructure()
	code := m.Run()
	os.Exit(code)
}

func ResetTestDB() {
	ResetDBStructure()
	InitDBStructure()
}
