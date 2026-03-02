#!/usr/bin/env bash
set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
  echo "SKIP: docker not available"
  exit 0
fi

wait_for() {
  local url=$1
  for _ in $(seq 1 80); do
    if curl -fsS "$url" >/dev/null 2>&1; then return 0; fi
    sleep 2
  done
  echo "FAIL: timeout waiting $url"
  return 1
}

COOKIE_ADMIN=$(mktemp)
COOKIE_VIEWER=$(mktemp)
trap 'rm -f "$COOKIE_ADMIN" "$COOKIE_VIEWER"' EXIT

wait_for http://localhost:8080/healthz
wait_for http://localhost:8083/readyz
wait_for http://localhost:8085/readyz

curl -fsS -c "$COOKIE_ADMIN" -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"Email":"admin@sentinel.local","Password":"Sentinel#123"}' >/dev/null
curl -fsS -c "$COOKIE_VIEWER" -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"Email":"viewer@sentinel.local","Password":"Sentinel#123"}' >/dev/null

INCIDENT_ID=$(curl -fsS -b "$COOKIE_ADMIN" http://localhost:8085/queue | jq -r '.[0].id // empty')
if [ -z "$INCIDENT_ID" ]; then
  echo "SKIP: no queue incident available"
  exit 0
fi

CREATE=$(curl -s -b "$COOKIE_ADMIN" -X POST "http://localhost:8083/incidents/${INCIDENT_ID}/promote-case" -H 'Content-Type: application/json' -d '{"reason":"integration-case-test"}')
CASE_ID=$(echo "$CREATE" | jq -r '.case_id // empty')
if [ -z "$CASE_ID" ]; then
  # duplicate path expected in steady-state systems
  CASE_ID=$(echo "$CREATE" | jq -r '.existing_case_id // empty')
fi
if [ -z "$CASE_ID" ]; then
  echo "FAIL: could not resolve case id from create/duplicate response"
  exit 1
fi

STATUS=$(curl -s -o /tmp/case_conflict.json -w "%{http_code}" -b "$COOKIE_ADMIN" -X POST "http://localhost:8083/incidents/${INCIDENT_ID}/promote-case" -H 'Content-Type: application/json' -d '{"reason":"integration-case-test-duplicate"}')
[ "$STATUS" = "409" ]
jq -e '.existing_case_id != null' /tmp/case_conflict.json >/dev/null

curl -fsS -b "$COOKIE_ADMIN" -X PATCH "http://localhost:8083/cases/${CASE_ID}" -H 'Content-Type: application/json' -d '{"status":"investigating","owner":"analyst@sentinel.local"}' >/dev/null
curl -fsS -b "$COOKIE_ADMIN" -X POST "http://localhost:8083/cases/${CASE_ID}/notes" -H 'Content-Type: application/json' -d '{"note":"integration note"}' >/dev/null

curl -fsS -b "$COOKIE_ADMIN" "http://localhost:8083/cases/${CASE_ID}" > /tmp/case_detail.json
jq -e '.actions | length >= 2' /tmp/case_detail.json >/dev/null
jq -e '.actions[0].id <= .actions[-1].id' /tmp/case_detail.json >/dev/null

FORBID=$(curl -s -o /dev/null -w "%{http_code}" -b "$COOKIE_VIEWER" -X PATCH "http://localhost:8083/cases/${CASE_ID}" -H 'Content-Type: application/json' -d '{"status":"closed"}')
[ "$FORBID" = "403" ]

echo "PASS: case workflow regression checks"
