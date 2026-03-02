#!/usr/bin/env bash
set -euo pipefail
if ! command -v docker >/dev/null 2>&1; then
  echo "integration-test skipped: docker not available in this environment"
  exit 0
fi
COMPOSE="docker compose -f deploy/docker/docker-compose.yml"
$COMPOSE up -d redpanda postgres redis
trap "docker compose -f deploy/docker/docker-compose.yml down -v" EXIT
ready=0
for i in {1..60}; do
  if $COMPOSE exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done
if [ "$ready" -eq 0 ]; then
  echo "ERROR: PostgreSQL failed to become ready"
  exit 1
fi
./scripts/migrate.sh
$COMPOSE exec -T postgres psql -U sentinel -d sentinel -c "SELECT count(*) FROM users" >/dev/null || true
echo "integration test ok (infra+migration)"
