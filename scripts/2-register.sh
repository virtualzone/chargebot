#!/bin/sh
TESLA_API_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Ik1GUWpMaVF4OEZEeEdka2l1VDhuOW5RRGNRNCJ9.eyJndHkiOiJjbGllbnQtY3JlZGVudGlhbHMiLCJzdWIiOiIxOTJiMzBjZS1mMTZlLTQ2ZTEtOWQ3Yy1kNDAzYjczYWY3YjkiLCJpc3MiOiJodHRwczovL2F1dGgudGVzbGEuY29tL29hdXRoMi92My9udHMiLCJhenAiOiJlOTk0MWYwOGUwZDYtNGMyZi1iOGVlLTI5MTA2MGVjNjQ4YSIsImF1ZCI6WyJodHRwczovL2F1dGgudGVzbGEuY29tL29hdXRoMi92My9jbGllbnRpbmZvIiwiaHR0cHM6Ly9mbGVldC1hcGkucHJkLmV1LnZuLmNsb3VkLnRlc2xhLmNvbSJdLCJleHAiOjE3MDA0MTIzNzQsImlhdCI6MTcwMDM4MzU3NCwic2NwIjpbInZlaGljbGVfZGV2aWNlX2RhdGEiLCJ2ZWhpY2xlX2NtZHMiLCJ2ZWhpY2xlX2NoYXJnaW5nX2NtZHMiLCJvcGVuaWQiXX0.Jtl4VLdGg-Convr-NzXW6oSNQ1tjghLNY-Txd_Nj3KMinX9Z9zath5sBqV_HsjEv4NXQ2oKWBd342cjiOdBsZ7L5_K_Eluu_WgTrkO-ZDlZHCgQJHabycgIqqyziys9iZhxFspMwBWxMQHEVpiWlWrAJWn61ay1Ro1dWPRLuIPkEyL3J1FQu_M_u0XEgSXz3uKwAhWjJvQDHON9MAzA0L668xoReBObe8skfO4HrG_VEX8Y9OjfbjFjnuXpMVIo8dpHQZRr0yGzQ5E1vwjArPFPeoT60hni6dxm4pwlZ9HaECi4vOH-xtSkNOIAMRBPpG8KAaIRJq1lztM8GQ8tNyQ"
AUDIENCE="https://fleet-api.prd.eu.vn.cloud.tesla.com"
# Partner authentication token request
curl -v --header 'Content-Type: application/json' \
  --header "Authorization: Bearer $TESLA_API_TOKEN" \
  --data '{"domain":"tgc.virtualzone.de"}' \
  "${AUDIENCE}/api/1/partner_accounts"