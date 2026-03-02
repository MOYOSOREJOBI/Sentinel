#!/usr/bin/env bash
set -euo pipefail

ROOT=${1:-"$HOME/Desktop/fast"}
MODE=${MODE:-default}
COMPOSE_FILE="$ROOT/deploy/docker/docker-compose.yml"
FAST_FILE="$ROOT/deploy/docker/docker-compose.fast.yml"

if [ ! -d "$ROOT" ]; then
  echo "[error] repo root not found: $ROOT"
  echo "Usage: $0 [repo_root]"
  exit 1
fi

if [ ! -f "$COMPOSE_FILE" ]; then
  echo "[error] compose file not found: $COMPOSE_FILE"
  exit 1
fi

cd "$ROOT"

dc() {
  if [ "$MODE" = "fast" ] && [ -f "$FAST_FILE" ]; then
    docker compose -f "$COMPOSE_FILE" -f "$FAST_FILE" "$@"
  else
    docker compose -f "$COMPOSE_FILE" "$@"
  fi
}

start_docker_if_needed() {
  if docker system info >/dev/null 2>&1; then
    return 0
  fi

  if [ "$(uname -s)" = "Darwin" ]; then
    echo "=== STARTING DOCKER DESKTOP ==="
    open -a Docker || true
  fi

  echo "=== WAITING FOR DOCKER DAEMON ==="
  for i in $(seq 1 90); do
    if docker system info >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done

  echo "[error] docker daemon is not reachable"
  docker context show || true
  exit 1
}

open_url() {
  local url=$1
  if [ "$(uname -s)" = "Darwin" ]; then
    open -a "Google Chrome" "$url" 2>/dev/null || open "$url" 2>/dev/null || true
  else
    xdg-open "$url" >/dev/null 2>&1 || true
  fi
}

health_check() {
  local url=$1
  printf "%-35s" "$url"
  if curl -fsS --max-time 6 "$url" >/dev/null; then
    echo "OK"
  else
    echo "NO RESPONSE"
  fi
}

start_docker_if_needed

if [ -n "${DOCKER_HOST:-}" ]; then
  echo "=== UNSETTING DOCKER_HOST OVERRIDE ==="
  unset DOCKER_HOST
fi

echo "=== VALIDATE COMPOSE ==="
dc config >/dev/null

echo "=== CLEAN OLD STACK ==="
if grep -qE '^[[:space:]]*down:' Makefile 2>/dev/null; then
  make down || true
else
  dc down -v --remove-orphans || true
fi

echo "=== RUN DOCTOR ==="
make doctor || true

echo "=== START DEMO ($MODE) ==="
if [ "$MODE" = "fast" ]; then
  make demo-fast
else
  make demo
fi

echo "=== STATUS ==="
dc ps

echo "=== QUICK CHECKS ==="
health_check http://localhost:3000
health_check http://localhost:8080/healthz
health_check http://localhost:8085/readyz
health_check http://localhost:8083/readyz
health_check http://localhost:8084/readyz
health_check http://localhost:9090

echo "=== OPENING KEY PAGES IN BROWSER ==="
open_url http://localhost:3000/command-center
open_url http://localhost:3000/queue
open_url http://localhost:3000/trust
open_url http://localhost:3000/governance
open_url http://localhost:3000/executive

cat <<'MSG'

Login credentials:
  admin@sentinel.local / Sentinel#123

If startup fails, run:
  docker compose -f deploy/docker/docker-compose.yml ps
  docker compose -f deploy/docker/docker-compose.yml logs postgres --tail=200
  docker compose -f deploy/docker/docker-compose.yml logs query --tail=120
  docker compose -f deploy/docker/docker-compose.yml logs web --tail=120
MSG
