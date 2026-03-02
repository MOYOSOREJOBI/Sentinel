#!/usr/bin/env bash
set -euo pipefail

TS_DIR="${TS_DIR:-docs/audit/_latest}"
PROOF_DIR="${TS_DIR}/proofs"
LOG_DIR="${TS_DIR}/logs"
mkdir -p "${PROOF_DIR}" "${LOG_DIR}"
export GOCACHE="${GOCACHE:-/tmp/go-build-sentinel}"
mkdir -p "${GOCACHE}"

emit_status() {
  echo "STATUS=$1"
  echo "REASON=$2"
  echo "PROOFS_DIR=${PROOF_DIR}"
  echo "LOGS_DIR=${LOG_DIR}"
}

echo "=== GATE: Repo checks ==="
make doctor
make lint
make test
touch "${PROOF_DIR}/S1_baseline.ok"

echo "=== GATE: i18n consistency ==="
(
  cd web
  npm run check:i18n
) 2>&1 | tee "${LOG_DIR}/i18n-check.log"
touch "${PROOF_DIR}/S7_i18n.ok"

echo "=== GATE: security truthfulness checks ==="
go test ./internal/middleware ./cmd/gateway-api -count=1 2>&1 | tee "${LOG_DIR}/security-tests.log"
touch "${PROOF_DIR}/S8_security_truth.ok"

echo "=== GATE: Runtime proof ==="
set +e
TS_DIR="${TS_DIR}" ./scripts/runtime-proof.sh 2>&1 | tee "${LOG_DIR}/runtime-proof.log"
RP_EC=${PIPESTATUS[0]}
set -e
if [ "${RP_EC}" -eq 2 ]; then
  emit_status "SKIP" "docker unavailable"
  exit 2
fi
if [ "${RP_EC}" -ne 0 ]; then
  emit_status "FAIL" "runtime proof failed"
  exit 1
fi

echo "=== GATE: replay parity proof ==="
./integration/replay_equivalence_test.sh 2>&1 | tee "${LOG_DIR}/replay-parity.log"

echo "=== GATE: Playwright (Docker compose service) ==="
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  ./scripts/playwright-docker.sh
else
  echo "SKIP: Docker missing; Playwright cannot be proven." | tee -a "${LOG_DIR}/playwright.log"
  echo "docker unavailable" > "${PROOF_DIR}/SKIP_PLAYWRIGHT_DOCKER.txt"
  emit_status "SKIP" "docker unavailable for playwright"
  exit 2
fi

echo "PASS: gate complete"
emit_status "PASS" "gate complete"
