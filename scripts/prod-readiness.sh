#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="docker compose -f $ROOT/deploy/docker/docker-compose.yml"

wait_for() {
  local url="$1"
  for _ in $(seq 1 60); do
    if curl --max-time 5 -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "FAIL: timeout waiting for $url"
  exit 1
}

login_and_capture() {
  local email="$1"
  local response
  response="$(curl --max-time 5 -i -sS -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' --data "{\"Email\":\"${email}\",\"Password\":\"Sentinel#123\"}")"
  local token csrf
  token="$(COOKIE_BLOB="$response" python3 - <<'PY'
import os
blob = os.environ.get("COOKIE_BLOB", "")
for line in blob.splitlines():
    if not line.lower().startswith("set-cookie:"):
        continue
    raw = line.split(":", 1)[1].strip()
    key, _, rest = raw.partition("=")
    if key.strip() == "sentinel_token":
        print(rest.split(";", 1)[0])
        break
PY
)"
  csrf="$(COOKIE_BLOB="$response" python3 - <<'PY'
import os
blob = os.environ.get("COOKIE_BLOB", "")
for line in blob.splitlines():
    if not line.lower().startswith("set-cookie:"):
        continue
    raw = line.split(":", 1)[1].strip()
    key, _, rest = raw.partition("=")
    if key.strip() == "sentinel_csrf":
        print(rest.split(";", 1)[0])
        break
PY
)"
  if [ -z "$token" ] || [ -z "$csrf" ]; then
    return 1
  fi
  printf 'sentinel_token=%s; sentinel_csrf=%s\n%s\n' "$token" "$csrf" "$csrf"
}

wait_for http://localhost:8080/healthz
wait_for http://localhost:8083/readyz
wait_for http://localhost:8084/readyz
wait_for http://localhost:8085/readyz

if ! admin="$(login_and_capture admin@sentinel.local)"; then
  POSTGRES_URL=postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable go run "$ROOT/scripts/seed-users.go" >/dev/null
  admin="$(login_and_capture admin@sentinel.local)"
fi
viewer="$(login_and_capture viewer@sentinel.local)"

ADMIN_COOKIE="$(printf '%s' "$admin" | sed -n '1p')"
ADMIN_CSRF="$(printf '%s' "$admin" | sed -n '2p')"
VIEWER_COOKIE="$(printf '%s' "$viewer" | sed -n '1p')"
VIEWER_CSRF="$(printf '%s' "$viewer" | sed -n '2p')"

UNAUTH_ACK="$(curl --max-time 5 -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8083/alerts/1/ack)"
[ "$UNAUTH_ACK" = "401" ] || { echo "FAIL: unauth ack expected 401 got $UNAUTH_ACK"; exit 1; }

VIEWER_ACK="$(curl --max-time 5 -s -o /dev/null -w '%{http_code}' -H "Cookie: $VIEWER_COOKIE" -H "X-CSRF-Token: $VIEWER_CSRF" -X POST http://localhost:8083/alerts/1/ack)"
[ "$VIEWER_ACK" = "403" ] || { echo "FAIL: viewer ack expected 403 got $VIEWER_ACK"; exit 1; }

SEED_JSON="$(curl --max-time 5 -fsS -H "Cookie: $ADMIN_COOKIE" http://localhost:8085/debug/seed-status)"
python3 - "$SEED_JSON" <<'PY'
import json, sys
payload = json.loads(sys.argv[1])
if payload.get("scores", 0) <= 0:
    raise SystemExit("scores not warm")
PY

curl --max-time 5 -fsS -H "Cookie: $ADMIN_COOKIE" -H "X-CSRF-Token: $ADMIN_CSRF" -X POST http://localhost:8084/forecast/refresh >/tmp/sentinel-forecast-refresh.json
curl --max-time 5 -fsS -H "Cookie: $ADMIN_COOKIE" http://localhost:8085/forecast >/tmp/sentinel-forecast.json

METRICS="$(curl --max-time 5 -fsS http://localhost:8083/metrics)"
echo "$METRICS" | grep -q 'http_requests_total' || { echo "FAIL: http_requests_total missing"; exit 1; }
echo "$METRICS" | grep -q 'http_request_duration_seconds_bucket' || { echo "FAIL: http_request_duration_seconds missing"; exit 1; }

$COMPOSE ps >/tmp/sentinel-compose-ps.txt

echo "PASS: prod readiness checks"
