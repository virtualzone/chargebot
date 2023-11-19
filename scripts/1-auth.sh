#!/bin/sh
CLIENT_ID=e9941f08e0d6-4c2f-b8ee-291060ec648a
CLIENT_SECRET="ta-secret.v&aaZvfZ+POaQxcM"
AUDIENCE="https://fleet-api.prd.eu.vn.cloud.tesla.com"
# Partner authentication token request
curl -v --request POST \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode 'grant_type=client_credentials' \
  --data-urlencode "client_id=$CLIENT_ID" \
  --data-urlencode "client_secret=$CLIENT_SECRET" \
  --data-urlencode 'scope=openid vehicle_device_data vehicle_cmds vehicle_charging_cmds' \
  --data-urlencode "audience=$AUDIENCE" \
  'https://auth.tesla.com/oauth2/v3/token'