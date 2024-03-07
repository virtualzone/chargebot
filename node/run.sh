#!/bin/sh
TOKEN=12345 TELEMETRY_ENDPOINT=ws://localhost:8080/api/1/user/{token}/ws go run `ls *.go | grep -v _test.go`