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

login_cookie() {
  local response
  response="$(curl --max-time 5 -i -sS -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' --data '{"Email":"admin@sentinel.local","Password":"Sentinel#123"}')"
  local token csrf
  token="$(COOKIE_BLOB="$response" python3 - <<'PY'
import os
for line in os.environ.get("COOKIE_BLOB", "").splitlines():
    if line.lower().startswith("set-cookie: sentinel_token="):
        print(line.split("=", 1)[1].split(";", 1)[0]); break
PY
)"
  csrf="$(COOKIE_BLOB="$response" python3 - <<'PY'
import os
for line in os.environ.get("COOKIE_BLOB", "").splitlines():
    if line.lower().startswith("set-cookie: sentinel_csrf="):
        print(line.split("=", 1)[1].split(";", 1)[0]); break
PY
)"
  [ -n "$token" ] && [ -n "$csrf" ] || {
    POSTGRES_URL=postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable go run -tags seedusers "$ROOT/scripts/seed-users.go" >/dev/null
    login_cookie
    return
  }
  printf 'sentinel_token=%s; sentinel_csrf=%s\n' "$token" "$csrf"
}

run_load() {
  local name="$1"
  local url="$2"
  local outfile
  outfile="$(mktemp /tmp/${name}.XXXXXX)"
  export PERF_COOKIE
  export PERF_URL="$url"
  seq 1 250 | xargs -n1 -P10 sh -c '
    curl --max-time 5 -sS -o /dev/null -w "%{time_total}\n" -H "Cookie: ${PERF_COOKIE}" "${PERF_URL}" >> "'"$outfile"'"
  ' >/dev/null 2>&1
  python3 - "$name" "$outfile" <<'PY'
import math, sys
name, path = sys.argv[1], sys.argv[2]
vals = [float(x.strip()) for x in open(path) if x.strip()]
vals.sort()
if not vals:
    raise SystemExit(f"{name}: no samples")
idx = max(0, math.ceil(len(vals) * 0.95) - 1)
print(f"{name}_count={len(vals)}")
print(f"{name}_p95_ms={vals[idx]*1000:.2f}")
print(f"{name}_avg_ms={(sum(vals)/len(vals))*1000:.2f}")
if name == "queue" and vals[idx] > 0.5:
    raise SystemExit(f"{name} p95 {vals[idx]*1000:.2f}ms exceeds 500ms")
PY
  rm -f "$outfile"
}

memory_for() {
  local service="$1"
  local limit_mb="$2"
  local raw used unit mb
  raw="$($COMPOSE ps -q "$service" | xargs docker stats --no-stream --format '{{.MemUsage}}' | head -n1)"
  used="$(printf '%s' "$raw" | awk -F'/' '{print $1}' | xargs)"
  unit="${used##*[0-9.]}"
  case "$unit" in
    GiB) mb="$(python3 - <<PY
print(float("${used%GiB}") * 1024)
PY
)";;
    MiB) mb="${used%MiB}";;
    KiB) mb="$(python3 - <<PY
print(float("${used%KiB}") / 1024)
PY
)";;
    *) mb="0";;
  esac
  printf '%s_rss_mb=%.2f\n' "$service" "$mb"
  python3 - <<PY
value = float("${mb}")
limit = float("${limit_mb}")
service = "${service}"
assert value <= limit, f"{service} RSS {value:.2f}MB exceeds {limit:.0f}MB"
PY
}

wait_for http://localhost:8080/healthz
wait_for http://localhost:8085/readyz
wait_for http://localhost:3000/login

PERF_COOKIE="$(login_cookie)"

# Measure the query API directly for the server-side p95 budget; the web container
# is still checked separately for memory pressure.
run_load queue "http://localhost:8085/queue?window=24h"
run_load command_center "http://localhost:8085/command-center?window=24h"
run_load feed "http://localhost:8085/feed?window=24h"

memory_for web 600
memory_for query 800

echo "PASS: perf smoke"
