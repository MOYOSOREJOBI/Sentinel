#!/usr/bin/env bash
set -euo pipefail
COMPOSE="docker compose -f deploy/docker/docker-compose.yml"
if ! command -v docker >/dev/null 2>&1; then
  echo "SKIP: docker not available"
  exit 0
fi
$COMPOSE up -d --build query
trap '$COMPOSE down -v' EXIT
$COMPOSE stop postgres || true
sleep 3
curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/readyz | grep 503
$COMPOSE stop redpanda || true
curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/readyz | grep 503 || true
$COMPOSE stop redis || true
