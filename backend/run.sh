#!/bin/sh
DOMAIN=localhost:8080 DEV_PROXY=1 go run `ls *.go | grep -v _test.go`