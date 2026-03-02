#!/usr/bin/env bash
set -euo pipefail
compose="docker compose -f deploy/docker/docker-compose.yml"
ready=0
for i in {1..90}; do
  if $compose exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done
if [ "$ready" -eq 0 ]; then
  echo "ERROR: PostgreSQL failed to become ready after 90 seconds"
  exit 1
fi
echo "PostgreSQL is ready, running migrations..."
for f in sql/migrations/*.sql; do
  echo "Applying $(basename "$f")..."
  if grep -q '^-- +goose Up' "$f"; then
    awk '
      /^-- \+goose Up/ {in_up=1; next}
      /^-- \+goose Down/ {in_up=0}
      in_up {print}
    ' "$f" | $compose exec -T postgres psql -U sentinel -d sentinel -v ON_ERROR_STOP=1 -f -
  else
    $compose exec -T postgres psql -U sentinel -d sentinel -v ON_ERROR_STOP=1 -f - < "$f"
  fi
done
echo "Migrations complete."
