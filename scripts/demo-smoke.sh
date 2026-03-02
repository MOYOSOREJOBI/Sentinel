#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-deploy/docker/docker-compose.yml}"
DC="docker compose -f ${COMPOSE_FILE}"

root_check() {
  local name="$1" url="$2"
  local body
  body="$(curl -fsS "$url")"
  python3 - "$name" <<'PY' <<<"$body"
import json,sys
name=sys.argv[1]
obj=json.load(sys.stdin)
links=set(obj.get("links",[]))
required={"/healthz","/readyz","/metrics","/docs"}
if obj.get("service")!=name or not required.issubset(links):
    raise SystemExit(f"invalid root payload for {name}: {obj}")
print(f"ok root {name}")
PY
}

echo "=== DEMO SMOKE: reset and bring stack up ==="
make dev-keys
${DC} down -v --remove-orphans || true
${DC} up -d --build

echo "=== DEMO SMOKE: wait for healthy state ==="
deadline=$((SECONDS+240))
while true; do
  json="$(${DC} ps --format json)"
  if python3 - <<'PY' <<<"$json"
import json,sys
rows=json.load(sys.stdin)
long_running=[r for r in rows if r.get('Service') not in {'migrate','topics-init'}]
bad=[r for r in long_running if r.get('Health') and r.get('Health')!='healthy']
if bad:
    print('bad',bad)
    raise SystemExit(1)
print('healthy')
PY
  then
    break
  fi
  if [ "$SECONDS" -gt "$deadline" ]; then
    echo "FAIL: services not healthy in time"
    ${DC} ps
    exit 1
  fi
  sleep 4
done

echo "=== DEMO SMOKE: compose ps ==="
${DC} ps

echo "=== DEMO SMOKE: readiness probes ==="
for p in 8080 8085 8083 8084 8081 8082 8086; do
  curl -fsS "http://localhost:${p}/readyz" >/dev/null
  echo "readyz ok :${p}"
done
curl -fsS http://localhost:8090/readyz >/dev/null || curl -fsS http://localhost:8090/healthz >/dev/null

echo "=== DEMO SMOKE: root route JSON contracts ==="
root_check gateway-api http://localhost:8080/
root_check query http://localhost:8085/
root_check alerts http://localhost:8083/
root_check governance http://localhost:8084/
root_check inference http://localhost:8090/

echo "=== DEMO SMOKE: seed/data flow status ==="
for _ in $(seq 1 30); do
  body="$(curl -fsS http://localhost:8085/debug/seed-status || true)"
  if [ -n "$body" ] && python3 - <<'PY' <<<"$body"
import json,sys
obj=json.load(sys.stdin)
req=["raw_ticks","candles","features","scores","alerts","incidents"]
raise SystemExit(0 if all(int(obj.get(k,0))>0 for k in req) else 1)
PY
  then
    echo "seed-status ready: $body"
    break
  fi
  sleep 2
done

echo "=== DEMO SMOKE: SSE endpoint ==="
headers="$(curl -fsSI http://localhost:3000/api/sse/alerts)"
echo "$headers" | rg -i "content-type: text/event-stream" >/dev/null

echo "=== DEMO SMOKE: web ==="
curl -fsS -D- http://localhost:3000/ | head -n 30

echo "=== DEMO SMOKE: Prometheus up==1 ==="
python3 scripts/prom_up_check.py http://localhost:9090 gateway query alerts governance aggregator features inference

echo "=== DEMO SMOKE: Grafana provisioning ==="
curl -fsS -u admin:sentinel http://localhost:3001/api/datasources | python3 - <<'PY'
import json,sys
arr=json.load(sys.stdin)
if not any(d.get('type')=='prometheus' for d in arr):
    raise SystemExit('prometheus datasource not provisioned')
print('datasource ok')
PY
curl -fsS -u admin:sentinel http://localhost:3001/api/search?query=sentinel | python3 - <<'PY'
import json,sys
arr=json.load(sys.stdin)
if not arr:
    raise SystemExit('no dashboards found')
print('dashboard search ok', len(arr))
PY

echo "PASS: demo smoke complete"
