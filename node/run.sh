#!/bin/sh
DEV_PROXY=1 CRYPT_KEY=12345678901234567890123456789012 TOKEN=12345 TELEMETRY_ENDPOINT=ws://localhost:8080/api/1/user/{token}/ws go run `ls *.go | grep -v _test.go`