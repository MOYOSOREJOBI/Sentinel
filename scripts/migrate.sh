#!/usr/bin/env bash
set -euo pipefail
compose="docker compose -f deploy/docker/docker-compose.yml"
PG_USER="${POSTGRES_USER:-sentinel}"
PG_DB="${POSTGRES_DB:-sentinel}"

ready=0
for i in {1..90}; do
  if $compose exec -T postgres pg_isready -U "$PG_USER" -d "$PG_DB" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done
if [ "$ready" -eq 0 ]; then
  echo "ERROR: PostgreSQL failed to become ready after 90 seconds"
  exit 1
fi

$compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -c "CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())" >/dev/null

applied_count="$($compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -At -c "SELECT count(*) FROM schema_migrations")"
if [ "$applied_count" = "0" ]; then
  has_existing_schema="$($compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -At -c "SELECT CASE WHEN to_regclass('public.users') IS NULL THEN 0 ELSE 1 END")"
  if [ "$has_existing_schema" = "1" ]; then
    for baseline in \
      001_init.sql \
      002_indexes.sql \
      003_incidents_cases_lineage.sql \
      004_replay_jobs.sql \
      005_model_deployments.sql \
      006_watchlists_and_preferences.sql \
      007_escalation_oncall.sql \
      008_scores_lineage.sql \
      009_incident_ops.sql \
      010_timescale_rollups.sql \
      011_preferred_locale.sql
    do
      $compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations(name) VALUES ('$baseline') ON CONFLICT DO NOTHING" >/dev/null
    done
  fi
fi

echo "PostgreSQL is ready, running migrations..."
for f in sql/migrations/*.sql; do
  name="$(basename "$f")"
  already_applied="$($compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -At -c "SELECT 1 FROM schema_migrations WHERE name='$name' LIMIT 1")"
  if [ "$already_applied" = "1" ]; then
    echo "Skipping $name (already applied)..."
    continue
  fi
  echo "Applying $name..."
  if grep -q '^-- +goose Up' "$f"; then
    awk '
      /^-- \+goose Up/ {in_up=1; next}
      /^-- \+goose Down/ {in_up=0}
      in_up {print}
    ' "$f" | $compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -f -
  else
    $compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -f - < "$f"
  fi
  $compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations(name) VALUES ('$name') ON CONFLICT DO NOTHING" >/dev/null
done
echo "Migrations complete."
