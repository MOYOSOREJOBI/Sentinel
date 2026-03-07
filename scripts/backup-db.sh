#!/usr/bin/env bash
# Backup or restore the Sentinel PostgreSQL database.
#
# Usage:
#   ./scripts/backup-db.sh backup              # creates backups/sentinel-<timestamp>.sql.gz
#   ./scripts/backup-db.sh restore <file.gz>   # restores from a backup file
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="docker compose -f $ROOT/deploy/docker/docker-compose.yml"
BACKUP_DIR="$ROOT/backups"
mkdir -p "$BACKUP_DIR"

POSTGRES_USER="${POSTGRES_USER:-sentinel}"
POSTGRES_DB="${POSTGRES_DB:-sentinel}"

cmd="${1:-backup}"

case "$cmd" in
  backup)
    TS="$(date -u +%Y%m%dT%H%M%SZ)"
    OUT="$BACKUP_DIR/sentinel-${TS}.sql.gz"
    echo "[backup] dumping $POSTGRES_DB → $OUT"
    $COMPOSE exec -T postgres pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" | gzip > "$OUT"
    SIZE="$(du -sh "$OUT" | cut -f1)"
    echo "[backup] done: $OUT ($SIZE)"

    # Keep only last 10 backups
    ls -t "$BACKUP_DIR"/sentinel-*.sql.gz 2>/dev/null | tail -n +11 | xargs -r rm --
    echo "[backup] old backups pruned (keeping 10 most recent)"
    ;;

  restore)
    FILE="${2:-}"
    if [ -z "$FILE" ]; then
      echo "Usage: $0 restore <backup-file.sql.gz>"
      exit 1
    fi
    if [ ! -f "$FILE" ]; then
      echo "ERROR: file not found: $FILE"
      exit 1
    fi
    echo "[restore] WARNING: this will DROP and recreate the $POSTGRES_DB database."
    read -rp "Type 'yes' to continue: " confirm
    [ "$confirm" = "yes" ] || { echo "aborted"; exit 1; }
    echo "[restore] restoring from $FILE"
    $COMPOSE exec -T postgres psql -U "$POSTGRES_USER" -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$POSTGRES_DB' AND pid <> pg_backend_pid();" postgres >/dev/null
    $COMPOSE exec -T postgres dropdb -U "$POSTGRES_USER" --if-exists "$POSTGRES_DB"
    $COMPOSE exec -T postgres createdb -U "$POSTGRES_USER" "$POSTGRES_DB"
    gunzip -c "$FILE" | $COMPOSE exec -T postgres psql -U "$POSTGRES_USER" "$POSTGRES_DB"
    echo "[restore] done"
    ;;

  *)
    echo "Usage: $0 {backup|restore <file>}"
    exit 1
    ;;
esac
