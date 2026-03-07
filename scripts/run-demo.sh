#!/usr/bin/env bash
set -euo pipefail
mode=${1:-default}
use_local_override=${SENTINEL_USE_LOCAL_OVERRIDE:-0}

port_in_use() {
  local port=$1
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
    return $?
  fi
  return 1
}

any_ports_in_use() {
  local port
  for port in "$@"; do
    if port_in_use "$port"; then
      return 0
    fi
  done
  return 1
}

choose_port() {
  local preferred=$1 fallback=$2
  if ! port_in_use "$preferred"; then
    echo "$preferred"
    return 0
  fi
  if ! port_in_use "$fallback"; then
    echo "$fallback"
    return 0
  fi
  local candidate
  for offset in $(seq 1 50); do
    candidate=$((fallback + offset))
    if ! port_in_use "$candidate"; then
      echo "$candidate"
      return 0
    fi
  done
  echo "ERROR: no open host port found near $preferred/$fallback" >&2
  exit 1
}

wait_for() {
  local name=$1 url=$2 max=${3:-60}
  echo "[wait] $name -> $url"
  for i in $(seq 1 "$max"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "[ok] $name"
      return 0
    fi
    sleep 2
  done
  echo "[fail] $name did not become ready: $url"
  return 1
}

if ! command -v docker >/dev/null 2>&1; then
  echo "SKIP: demo-smoke skipped because docker is not available in this environment"
  exit 0
fi
make dev-keys
if [ "$mode" = "fast" ]; then
  COMPOSE="docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.fast.yml"
else
  COMPOSE="docker compose -f deploy/docker/docker-compose.yml"
fi

export REDPANDA_PORT
export POSTGRES_PORT
export REDIS_PORT
export GATEWAY_PORT
export AGGREGATOR_PORT
export FEATURES_PORT
export INFERENCE_PORT
export ALERTS_PORT
export GOVERNANCE_PORT
export QUERY_PORT
export WEB_PORT
export PROMETHEUS_PORT
export GRAFANA_PORT

default_conflicts=0
if any_ports_in_use 9092 5432 6379 8080 8081 8082 8090 8083 8084 8085 3000 9090 3001; then
  default_conflicts=1
fi

override_conflicts=0
if any_ports_in_use 59092 55432 56379 58080 58081 58082 58090 58083 58084 58085 33000 59090 33001; then
  override_conflicts=1
fi

if [ "$use_local_override" = "1" ] || { [ "$default_conflicts" = "1" ] && [ "$override_conflicts" = "0" ]; }; then
  COMPOSE="$COMPOSE -f deploy/docker/docker-compose.override.local.yml"
  REDPANDA_PORT=59092
  POSTGRES_PORT=55432
  REDIS_PORT=56379
  GATEWAY_PORT=58080
  AGGREGATOR_PORT=58081
  FEATURES_PORT=58082
  INFERENCE_PORT=58090
  ALERTS_PORT=58083
  GOVERNANCE_PORT=58084
  QUERY_PORT=58085
  WEB_PORT=33000
  PROMETHEUS_PORT=59090
  GRAFANA_PORT=33001
  echo "[compose] using deploy/docker/docker-compose.override.local.yml"
elif [ "$default_conflicts" = "1" ]; then
  REDPANDA_PORT=$(choose_port 9092 59092)
  POSTGRES_PORT=$(choose_port 5432 55432)
  REDIS_PORT=$(choose_port 6379 56379)
  GATEWAY_PORT=$(choose_port 8080 58080)
  AGGREGATOR_PORT=$(choose_port 8081 58081)
  FEATURES_PORT=$(choose_port 8082 58082)
  INFERENCE_PORT=$(choose_port 8090 58090)
  ALERTS_PORT=$(choose_port 8083 58083)
  GOVERNANCE_PORT=$(choose_port 8084 58084)
  QUERY_PORT=$(choose_port 8085 58085)
  WEB_PORT=$(choose_port 3000 33000)
  PROMETHEUS_PORT=$(choose_port 9090 59090)
  GRAFANA_PORT=$(choose_port 3001 33001)
else
  REDPANDA_PORT=9092
  POSTGRES_PORT=5432
  REDIS_PORT=6379
  GATEWAY_PORT=8080
  AGGREGATOR_PORT=8081
  FEATURES_PORT=8082
  INFERENCE_PORT=8090
  ALERTS_PORT=8083
  GOVERNANCE_PORT=8084
  QUERY_PORT=8085
  WEB_PORT=3000
  PROMETHEUS_PORT=9090
  GRAFANA_PORT=3001
fi

echo "[ports] web=$WEB_PORT gateway=$GATEWAY_PORT query=$QUERY_PORT postgres=$POSTGRES_PORT redis=$REDIS_PORT redpanda=$REDPANDA_PORT"

$COMPOSE up -d redpanda postgres redis
./scripts/migrate.sh
./scripts/create-topics.sh
$COMPOSE up -d --build gateway-api simulator aggregator features inference alerts governance query web prometheus grafana nginx
make seed POSTGRES_PORT="$POSTGRES_PORT"

wait_for gateway "http://localhost:${GATEWAY_PORT}/healthz"
wait_for query "http://localhost:${QUERY_PORT}/readyz"
wait_for alerts "http://localhost:${ALERTS_PORT}/readyz"
wait_for governance "http://localhost:${GOVERNANCE_PORT}/readyz"
wait_for command-center "http://localhost:${WEB_PORT}/command-center" 80

COOKIE_JAR=$(mktemp)
LOGIN_JSON='{"Email":"admin@sentinel.local","Password":"Sentinel#123"}'
curl -fsS -c "$COOKIE_JAR" -X POST "http://localhost:${GATEWAY_PORT}/auth/login" -H 'Content-Type: application/json' -d "$LOGIN_JSON" >/dev/null
CSRF=$(awk '/sentinel_csrf/ {print $7}' "$COOKIE_JAR" | tail -n1)
[ -n "$CSRF" ]
for ep in command-center queue trust world-map governance/summary executive-summary cases; do
  curl -fsS -b "$COOKIE_JAR" "http://localhost:${QUERY_PORT}/$ep" >/dev/null
  echo "[ok] query/$ep"
done
rm -f "$COOKIE_JAR"

echo "[pass] Sentinel stack ready ($mode)"
SKIPPED=0
BASE_URL="http://localhost:${WEB_PORT}" ./scripts/browser-validate.sh || rc=$?
if [ "${rc:-0}" -eq 3 ]; then
  echo "[skip] browser validation skipped"
  SKIPPED=1
elif [ "${rc:-0}" -ne 0 ]; then
  echo "[fail] browser validation"; exit 1
fi
unset rc
BASE_URL="http://localhost:${WEB_PORT}" ./scripts/capture-screenshots.sh || rc=$?
if [ "${rc:-0}" -eq 3 ]; then
  echo "[skip] screenshot capture skipped"
  SKIPPED=1
elif [ "${rc:-0}" -ne 0 ]; then
  echo "[fail] screenshot capture"; exit 1
fi
if [ "$SKIPPED" -eq 1 ]; then
  echo "SKIP: Sentinel Demo Ready with skipped browser artifact steps ($mode)"
else
  echo "PASS: Sentinel Demo Ready ($mode)"
fi
