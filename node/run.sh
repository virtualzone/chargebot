#!/bin/sh
if command -v sqlite3 -version &> /dev/null
then
    rm -f /tmp/chargebot_node.db
    INIT_DB_ONLY=1 go run `ls *.go | grep -v _test.go`
    sqlite3 /tmp/chargebot_node.db "insert into vehicles (vin, display_name, enabled, target_soc, max_amps, surplus_charging, min_surplus, min_chargetime, lowcost_charging, max_price, tibber_token) values ('5YJXCBE24KF152671', 'Model Y', 0, 0, 0, 0, 0, 0, 0, 0, '')"
    for i in $(seq 10 59); do sqlite3 /tmp/chargebot_node.db "insert into surpluses (ts, surplus_watts) values ('2006-01-02 15:${i}:05', $((-2000 + $RANDOM % 9000)))"; done
fi
MQTT_BROKER="tcp://broker.hivemq.com:1883" DEMO_MODE=1 PORT=8081 DEV_PROXY=1 CRYPT_KEY=12345678901234567890123456789012 TOKEN=f8f78ed5-204e-4e5b-a0cf-ceac819c3b2d PASSWORD=z2fxvdzMEOrqi1cB CMD_ENDPOINT="http://localhost:8080/api/1/user/{token}" TELEMETRY_ENDPOINT=ws://localhost:8080/api/1/user/{token}/ws go run `ls *.go | grep -v _test.go`