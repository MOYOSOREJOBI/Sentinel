#!/usr/bin/env bash
set -euo pipefail

TS="$(date -u +%Y%m%dT%H%M%SZ)"
TS_DIR="docs/audit/${TS}"
PROOF_DIR="${TS_DIR}/proofs"
LOG_DIR="${TS_DIR}/logs"
mkdir -p "${PROOF_DIR}" "${LOG_DIR}"

ln -sfn "${TS}" docs/audit/_latest

set +e
TS_DIR="${TS_DIR}" ./scripts/gate.sh 2>&1 | tee "${LOG_DIR}/gate.log"
GATE_EC=${PIPESTATUS[0]}
set -e

has() {
  [ -f "${PROOF_DIR}/$1" ] && echo 1 || echo 0
}

s2="$(has S2_compose_healthy.ok)"
s3="$(has S3_root_routes.ok)"
s4="$(has S4_prom_up.ok)"
s5="$(has S5_grafana_provisioned.ok)"
s6="$(has S6_data_warm.ok)"
s1="$(has S1_baseline.ok)"
s7="$(has S7_i18n.ok)"
s8="$(has S8_security_truth.ok)"
s9="$(has S9_replay_parity.ok)"
s10="$(has S10_playwright.ok)"

passed=$((s1+s2+s3+s4+s5+s6+s7+s8+s9+s10))
score=$((passed*100/10))
status="FAIL"
runtime="NONE"
reason="incomplete proofs"
if [ "${passed}" -eq 10 ]; then
  status="PASS"
  runtime="FULL"
  reason="all proofs present"
elif [ -f "${PROOF_DIR}/SKIP_DOCKER.txt" ] || [ "${GATE_EC}" -eq 2 ]; then
  status="SKIP"
  runtime="NONE"
  reason="docker unavailable"
fi

cat > "${TS_DIR}/RESULT.md" <<MD
COMPLETION: ${score}%
STATUS: ${status}
RUNTIME COVERAGE: ${runtime}
REASON: ${reason}

S1=${s1} S2=${s2} S3=${s3} S4=${s4} S5=${s5} S6=${s6} S7=${s7} S8=${s8} S9=${s9} S10=${s10}

GATE_EXIT=${GATE_EC}
PROOFS=${PROOF_DIR}
LOGS=${LOG_DIR}
STATUS_LINE=STATUS=${status}
REASON_LINE=REASON=${reason}
MD

cat > "${TS_DIR}/TRACEABILITY.md" <<MD
# Traceability

- ts_dir: ${TS_DIR}
- ran: TS_DIR=${TS_DIR} ./scripts/gate.sh
- scoring source: proof markers in ${PROOF_DIR}
- required markers:
  - S1_baseline.ok
  - S2_compose_healthy.ok
  - S3_root_routes.ok
  - S4_prom_up.ok
  - S5_grafana_provisioned.ok
  - S6_data_warm.ok
  - S7_i18n.ok
  - S8_security_truth.ok
  - S9_replay_parity.ok
  - S10_playwright.ok
- skips:
  - SKIP_DOCKER.txt indicates runtime not executed due missing Docker
  - SKIP_PLAYWRIGHT_DOCKER.txt indicates Playwright not executed due missing Docker
- logs: ${LOG_DIR}
MD

ln -sfn "${TS}" docs/audit/_latest
cat "${TS_DIR}/RESULT.md"
echo "STATUS=${status}"
echo "REASON=${reason}"
echo "PROOFS_DIR=${PROOF_DIR}"
echo "LOGS_DIR=${LOG_DIR}"

[ "${score}" -eq 100 ]
