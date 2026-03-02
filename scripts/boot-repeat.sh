#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="docker compose -f $ROOT/deploy/docker/docker-compose.yml"
RUNS="${1:-5}"

wait_for() {
  local url="$1"
  for _ in $(seq 1 90); do
    if curl --max-time 5 -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "FAIL: timeout waiting for $url"
  exit 1
}

cookie_line() {
  local response="$1"
  COOKIE_BLOB="$response" python3 - <<'PY'
import os
token = ""
csrf = ""
for line in os.environ.get("COOKIE_BLOB", "").splitlines():
    low = line.lower()
    if low.startswith("set-cookie: sentinel_token="):
        token = line.split("=", 1)[1].split(";", 1)[0]
    if low.startswith("set-cookie: sentinel_csrf="):
        csrf = line.split("=", 1)[1].split(";", 1)[0]
print(f"sentinel_token={token}; sentinel_csrf={csrf}")
PY
}

for i in $(seq 1 "$RUNS"); do
  echo "=== boot run $i/$RUNS ==="
  $COMPOSE down -v --remove-orphans >/dev/null 2>&1 || true
  $COMPOSE up -d --build >/dev/null
  wait_for http://localhost:8080/healthz
  wait_for http://localhost:8085/readyz
  POSTGRES_URL=postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable go run "$ROOT/scripts/seed-users.go" >/dev/null
  login="$(curl --max-time 5 -i -sS -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' --data '{"Email":"admin@sentinel.local","Password":"Sentinel#123"}')"
  cookie="$(cookie_line "$login")"
  deadline=$((SECONDS+90))
  while true; do
    if seed_json="$(curl --max-time 5 -fsS -H "Cookie: $cookie" http://localhost:8085/debug/seed-status 2>/dev/null)"; then
      if python3 - "$seed_json" <<'PY'
import json, sys
payload = json.loads(sys.argv[1])
required = ("raw_ticks", "candles", "features", "scores")
assert all(int(payload.get(k, 0)) > 0 for k in required), payload
PY
      then
        break
      fi
    fi
    if [ "$SECONDS" -gt "$deadline" ]; then
      echo "FAIL: warmup did not complete on run $i"
      exit 1
    fi
    sleep 2
  done
  echo "boot_run_${i}=ok"
done

echo "PASS: ${RUNS} deterministic boot cycles"
