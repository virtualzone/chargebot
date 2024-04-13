#!/bin/sh
if command -v sqlite3 -version &> /dev/null
then
    rm -f /tmp/chargebot.db
    INIT_DB_ONLY=1 go run `ls *.go | grep -v _test.go`
    sqlite3 /tmp/chargebot.db "insert into users (id, tesla_user_id) values ('8da4de32-8829-4a99-b5fe-2103e25be03b', 'bb1e0a53-a914-49a2-a939-b533eb05663a')"
    sqlite3 /tmp/chargebot.db "insert into api_tokens (token, user_id, passhash) values ('f8f78ed5-204e-4e5b-a0cf-ceac819c3b2d', '8da4de32-8829-4a99-b5fe-2103e25be03b', '56dca395afc313da732f78a8e2ef4059ac2260441c47f43f32642c732c19b814')"
fi
DOMAIN=localhost:8080 DEV_PROXY=1 go run `ls *.go | grep -v _test.go`