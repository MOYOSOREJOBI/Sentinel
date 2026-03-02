#!/usr/bin/env bash
set -euo pipefail
COMPOSE="docker compose -f deploy/docker/docker-compose.yml"
if ! command -v docker >/dev/null 2>&1; then
  echo "SKIP: docker not available"
  exit 0
fi
$COMPOSE up -d --build
trap '$COMPOSE down -v' EXIT
EVENT='{"event_id":"evt-idem-1","symbol":"AAPL","price":100.1,"volume":10,"event_time":"2026-01-01T00:00:00Z"}'
for _ in 1 2; do
  $COMPOSE exec -T redpanda rpk topic produce raw.ticks -k AAPL <<< "$EVENT" >/dev/null
  sleep 1
done
sleep 5
COUNT=$($COMPOSE exec -T postgres psql -U sentinel -d sentinel -Atc "select count(*) from raw_ticks where event_id='evt-idem-1'")
[ "$COUNT" = "1" ]
