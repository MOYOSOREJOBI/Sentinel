#!/usr/bin/env bash
set -euo pipefail
COMPOSE_FILE="${COMPOSE_FILE:-deploy/docker/docker-compose.yml}"
DC="docker compose -f ${COMPOSE_FILE}"
TS_DIR="${TS_DIR:-docs/audit/_latest}"
mkdir -p "${TS_DIR}/proofs" "${TS_DIR}/logs"
echo "=== Playwright via compose service ===" | tee -a "${TS_DIR}/logs/playwright.log"
${DC} --profile e2e run --rm playwright 2>&1 | tee -a "${TS_DIR}/logs/playwright.log"
touch "${TS_DIR}/proofs/S10_playwright.ok"
