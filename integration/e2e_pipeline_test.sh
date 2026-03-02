#!/usr/bin/env bash
set -euo pipefail
COMPOSE="docker compose -f deploy/docker/docker-compose.yml"
if ! command -v docker >/dev/null 2>&1; then
  echo "SKIP: docker not available"
  exit 0
fi
$COMPOSE up -d --build
trap '$COMPOSE down -v' EXIT
for i in {1..60}; do
  if curl -fsS http://localhost:8085/readyz >/dev/null; then break; fi
  sleep 2
done
curl -fsS http://localhost:8085/scores | jq '.[0] | has("symbol") and has("score") and has("severity")' | grep true
curl -fsS http://localhost:8085/alerts | jq '.[0] | has("symbol") and has("status")' | grep true
$COMPOSE exec -T postgres psql -U sentinel -d sentinel -c "select count(*)>0 from audit_log" | grep t
