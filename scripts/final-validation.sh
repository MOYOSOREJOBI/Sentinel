#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

COMPOSE="docker compose -f deploy/docker/docker-compose.yml"

$COMPOSE down -v --remove-orphans
make dev-keys
$COMPOSE up -d --build

for url in \
  http://localhost:8080/readyz \
  http://localhost:8085/readyz \
  http://localhost:8081/readyz \
  http://localhost:8082/readyz \
  http://localhost:8083/readyz \
  http://localhost:8084/readyz \
  http://localhost:8090/readyz

do
  echo "Waiting for $url"
  for i in {1..90}; do
    if curl -fsS "$url" >/dev/null; then
      echo "Ready: $url"
      break
    fi
    sleep 2
    if [ "$i" -eq 90 ]; then
      echo "ERROR: timeout waiting for $url"
      $COMPOSE logs --tail=200
      exit 1
    fi
  done
done

curl -fsS http://localhost:8080/healthz && echo
curl -fsS http://localhost:8080/readyz && echo
curl -fsS http://localhost:8085/alerts | head -c 500 && echo
curl -fsS http://localhost:8085/scores | head -c 500 && echo
curl -fsS http://localhost:9090/-/healthy && echo

make integration-suite

$COMPOSE restart postgres
sleep 5
curl -fsS http://localhost:8085/readyz && echo

$COMPOSE restart redpanda
sleep 5
curl -fsS http://localhost:8081/readyz && echo
curl -fsS http://localhost:8082/readyz && echo
curl -fsS http://localhost:8083/readyz && echo
curl -fsS http://localhost:8090/readyz && echo

$COMPOSE ps
$COMPOSE logs --tail=200

echo "ALL VALIDATIONS PASSED"
