#!/bin/sh
DOMAIN=localhost:8080 DEV_PROXY=1 CRYPT_KEY=12345678901234567890123456789012 go run `ls *.go | grep -v _test.go`