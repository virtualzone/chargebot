#!/bin/sh
PORT=8081 DEV_PROXY=1 CRYPT_KEY=12345678901234567890123456789012 TOKEN=f8f78ed5-204e-4e5b-a0cf-ceac819c3b2d PASSWORD=z2fxvdzMEOrqi1cB CMD_ENDPOINT="http://localhost:8080/api/1/user/{token}" TELEMETRY_ENDPOINT=ws://localhost:8080/api/1/user/{token}/ws go run `ls *.go | grep -v _test.go`