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

COOKIE_VIEWER=$(mktemp)
COOKIE_ANALYST=$(mktemp)
COOKIE_ADMIN=$(mktemp)
trap 'rm -f "$COOKIE_VIEWER" "$COOKIE_ANALYST" "$COOKIE_ADMIN"' EXIT

wait_for http://localhost:8080/healthz
wait_for http://localhost:8085/readyz

curl -fsS -c "$COOKIE_VIEWER" -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"Email":"viewer@sentinel.local","Password":"Sentinel#123"}' >/dev/null
curl -fsS -c "$COOKIE_ANALYST" -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"Email":"analyst@sentinel.local","Password":"Sentinel#123"}' >/dev/null
curl -fsS -c "$COOKIE_ADMIN" -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"Email":"admin@sentinel.local","Password":"Sentinel#123"}' >/dev/null

for c in "$COOKIE_VIEWER" "$COOKIE_ANALYST" "$COOKIE_ADMIN"; do
  curl -fsS -b "$c" http://localhost:8085/command-center >/dev/null
  curl -fsS -b "$c" http://localhost:8085/queue >/dev/null
  curl -fsS -b "$c" http://localhost:8085/world-map >/dev/null
  curl -fsS -b "$c" http://localhost:8085/trust >/dev/null
  curl -fsS -b "$c" http://localhost:8085/executive-summary >/dev/null
done

VIEWER_GOV=$(curl -s -o /dev/null -w "%{http_code}" -b "$COOKIE_VIEWER" http://localhost:8085/governance/summary)
ANALYST_GOV=$(curl -s -o /dev/null -w "%{http_code}" -b "$COOKIE_ANALYST" http://localhost:8085/governance/summary)
ADMIN_GOV=$(curl -s -o /dev/null -w "%{http_code}" -b "$COOKIE_ADMIN" http://localhost:8085/governance/summary)

[ "$VIEWER_GOV" = "403" ]
[ "$ANALYST_GOV" = "403" ]
[ "$ADMIN_GOV" = "200" ]

echo "PASS: query auth + governance RBAC"
