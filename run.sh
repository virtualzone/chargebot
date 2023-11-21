#!/bin/sh
HOSTNAME=tgc-dev.virtualzone.de go run `ls *.go | grep -v _test.go`