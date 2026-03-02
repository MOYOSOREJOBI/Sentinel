#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="docker compose -f $ROOT/deploy/docker/docker-compose.yml"

cleanup() {
  $COMPOSE down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT

$COMPOSE up -d postgres redpanda topics-init migrate

wait_for_exit_0() {
  local svc="$1"
  local cid status exit_code

  cid="$($COMPOSE ps -q "$svc" | head -n1)"

  if [ -z "$cid" ]; then
    echo "No container found for service: $svc"
    return 1
  fi

  for _ in $(seq 1 60); do
    status="$(docker inspect -f '{{.State.Status}}' "$cid" 2>/dev/null || true)"
    exit_code="$(docker inspect -f '{{.State.ExitCode}}' "$cid" 2>/dev/null || true)"

    if [ "$status" = "exited" ] && [ "$exit_code" = "0" ]; then
      echo "$svc completed successfully"
      return 0
    fi

    if [ "$status" = "exited" ] && [ "$exit_code" != "0" ]; then
      echo "$svc exited with code $exit_code"
      docker logs "$cid" || true
      return 1
    fi

    sleep 1
  done

  echo "$svc did not complete successfully in time"
  docker logs "$cid" || true
  return 1
}

wait_for_exit_0 migrate
wait_for_exit_0 topics-init

$COMPOSE up -d gateway-api query alerts simulator aggregator features governance inference >/dev/null

for _ in $(seq 1 45); do
  if curl -fsS http://localhost:8085/readyz >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

LOGS="$($COMPOSE logs --no-color simulator aggregator features alerts 2>/dev/null || true)"
if echo "$LOGS" | grep -qi 'unknown_topic'; then
  echo "FAIL: detected unknown topic on boot"
  exit 1
fi

for _ in $(seq 1 45); do
  if curl -fsS -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"Email":"admin@sentinel.local","Password":"Sentinel#123"}' -c /tmp/startup.cookie >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

for _ in $(seq 1 45); do
  if curl -fsS -b /tmp/startup.cookie http://localhost:8085/debug/seed-status >/tmp/startup-seed.json 2>/dev/null; then
    if python3 - <<'PY'
import json
from pathlib import Path
payload = json.loads(Path('/tmp/startup-seed.json').read_text())
assert payload.get('scores', 0) > 0
PY
    then
      break
    fi
  fi
  sleep 2
done

python3 - <<'PY'
import json
from pathlib import Path
payload = json.loads(Path('/tmp/startup-seed.json').read_text())
assert payload.get('scores', 0) > 0, payload
PY

$COMPOSE ps
