#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="docker compose -f $ROOT/deploy/docker/docker-compose.yml"
PROOF_DIR="$ROOT/docs/audit/_latest/proofs"
PROOF_FILE="$PROOF_DIR/S4_ml_real.ok"

if ! command -v docker >/dev/null 2>&1; then
  echo "SKIP: docker not available"
  exit 0
fi

mkdir -p "$PROOF_DIR"

$COMPOSE down -v >/dev/null 2>&1 || true
$COMPOSE up -d --build

for _ in $(seq 1 45); do
  if curl -fsS http://localhost:8090/readyz >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

for _ in $(seq 1 45); do
  scores_count="$($COMPOSE exec -T postgres psql -U sentinel -d sentinel -At -c "SELECT count(*) FROM scores;" | tr -d '[:space:]')"
  if [ "${scores_count:-0}" -gt 0 ] 2>/dev/null; then
    break
  fi
  sleep 2
done

scores_count="$($COMPOSE exec -T postgres psql -U sentinel -d sentinel -At -c "SELECT count(*) FROM scores;" | tr -d '[:space:]')"
real_model_count="$($COMPOSE exec -T postgres psql -U sentinel -d sentinel -At -c "SELECT count(*) FROM scores WHERE model_version NOT LIKE '%fallback%';" | tr -d '[:space:]')"
lineage_count="$($COMPOSE exec -T postgres psql -U sentinel -d sentinel -At -c "SELECT count(*) FROM scores WHERE produced_at IS NOT NULL AND coalesce(scoring_run_id::text,'') <> '' AND coalesce(artifact_hash,'') <> '' AND coalesce(model_artifact_hash,'') <> '';" | tr -d '[:space:]')"
readyz="$(curl -fsS http://localhost:8090/readyz)"

[ "${scores_count:-0}" -gt 0 ]
[ "${real_model_count:-0}" -gt 0 ]
[ "${lineage_count:-0}" -gt 0 ]

{
  echo "scores_count=$scores_count"
  echo "real_model_count=$real_model_count"
  echo "lineage_count=$lineage_count"
  echo "readyz=$readyz"
} > "$PROOF_FILE"

echo "PASS ml_scores_persist"
