#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-deploy/docker/docker-compose.yml}"
DC="docker compose -f ${COMPOSE_FILE}"

TS_DIR="${TS_DIR:-docs/audit/_latest}"
PROOF_DIR="${TS_DIR}/proofs"
LOG_DIR="${TS_DIR}/logs"
mkdir -p "${PROOF_DIR}" "${LOG_DIR}"
COOKIE_JAR=""

cleanup() {
  if [ -n "${COOKIE_JAR}" ] && [ -f "${COOKIE_JAR}" ]; then
    rm -f "${COOKIE_JAR}"
  fi
}
trap cleanup EXIT

emit_status() {
  echo "STATUS=$1"
  echo "REASON=$2"
  echo "PROOFS_DIR=${PROOF_DIR}"
  echo "LOGS_DIR=${LOG_DIR}"
}

if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
  echo "SKIP: docker unavailable" | tee -a "${LOG_DIR}/runtime-proof.log"
  echo "docker unavailable" > "${PROOF_DIR}/SKIP_DOCKER.txt"
  emit_status "SKIP" "docker unavailable"
  exit 2
fi

log() { tee -a "${LOG_DIR}/runtime-proof.log"; }

echo "=== clean up ===" | log
${DC} down -v --remove-orphans >>"${LOG_DIR}/compose_down.log" 2>&1 || true

echo "=== ensure jwt keys ===" | log
./scripts/dev-keys.sh >>"${LOG_DIR}/runtime-proof.log" 2>&1

echo "=== up ===" | log
${DC} up -d --build >>"${LOG_DIR}/compose_up.log" 2>&1

echo "=== wait healthy (robust) ===" | log
deadline=$((SECONDS+300))
while true; do
  bad=$(${DC} ps -q | while read -r cid; do
    st=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$cid")
    name=$(docker inspect -f '{{.Name}}' "$cid" | sed 's#^/##')
    if [ "$st" != "healthy" ] && [ "$st" != "running" ]; then
      echo "$name:$st"
    fi
  done | wc -l | tr -d ' ')
  if [ "${bad}" = "0" ]; then break; fi
  if [ "${SECONDS}" -gt "${deadline}" ]; then
    echo "FAIL: containers not healthy" | log
    ${DC} ps >>"${LOG_DIR}/compose_ps.log" 2>&1 || true
    ${DC} logs --tail=300 >>"${LOG_DIR}/compose_logs_tail.log" 2>&1 || true
    emit_status "FAIL" "containers not healthy"
    exit 1
  fi
  sleep 3
done

${DC} ps >>"${LOG_DIR}/compose_ps.log" 2>&1
touch "${PROOF_DIR}/S2_compose_healthy.ok"

echo "=== root/ready/metrics ===" | log
curl -fsS http://localhost:8080/ >/dev/null
curl -fsS http://localhost:8085/ >/dev/null
curl -fsS http://localhost:8083/ >/dev/null
curl -fsS http://localhost:8084/ >/dev/null
touch "${PROOF_DIR}/S3_root_routes.ok"

curl -fsS http://localhost:8080/readyz >/dev/null
curl -fsS http://localhost:8085/readyz >/dev/null
curl -fsS http://localhost:8083/readyz >/dev/null
curl -fsS http://localhost:8084/readyz >/dev/null
curl -fsS http://localhost:8080/metrics >/dev/null
curl -fsS http://localhost:8085/metrics >/dev/null
curl -fsS http://localhost:8083/metrics >/dev/null
curl -fsS http://localhost:8084/metrics >/dev/null

echo "=== seed demo users ===" | log
make seed >>"${LOG_DIR}/runtime-proof.log" 2>&1

COOKIE_JAR="$(mktemp /tmp/sentinel-runtime-cookie.XXXXXX)"
curl -fsS -c "${COOKIE_JAR}" \
  -H 'content-type: application/json' \
  -X POST http://localhost:8080/auth/login \
  --data '{"Email":"admin@sentinel.local","Password":"Sentinel#123"}' \
  >/dev/null

echo "=== web reachable ===" | log
curl -fsS http://localhost:3000/ >/dev/null

echo "=== data warm <= 90s ===" | log
deadline=$((SECONDS+90))
while true; do
  if curl -fsS -b "${COOKIE_JAR}" http://localhost:8085/debug/seed-status -o "${LOG_DIR}/seed-status.json" && python3 - "${LOG_DIR}/seed-status.json" <<'PY'
import json,sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    d=json.load(f)
req=["raw_ticks","candles","features","scores"]
ok=all(int(d.get(k,0))>0 for k in req)
sys.exit(0 if ok else 1)
PY
  then break; fi
  if [ "${SECONDS}" -gt "${deadline}" ]; then
    echo "FAIL: warm data not present" | log
    emit_status "FAIL" "warm data not present"
    exit 1
  fi
  sleep 2
done

echo "=== web auth + proxy data path ===" | log
curl -fsS -c "${COOKIE_JAR}" \
  -H 'content-type: application/json' \
  -X POST http://localhost:3000/api/proxy/gateway-api/auth/login \
  --data '{"Email":"admin@sentinel.local","Password":"Sentinel#123"}' \
  | tee "${LOG_DIR}/web-login.json" >/dev/null

curl -fsS -b "${COOKIE_JAR}" http://localhost:3000/api/proxy/gateway-api/me | tee "${LOG_DIR}/web-me.json" >/dev/null
python3 - "${LOG_DIR}/web-me.json" <<'PY'
import json,sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    data=json.load(f)
assert data.get("role") in {"admin","analyst","viewer"}, data
assert data.get("user") or data.get("email"), data
PY

curl -fsS -b "${COOKIE_JAR}" http://localhost:3000/api/proxy/query/command-center | tee "${LOG_DIR}/web-command-center.json" >/dev/null
python3 - "${LOG_DIR}/web-command-center.json" <<'PY'
import json,sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    data=json.load(f)
value = data.get("openIncidents", data.get("open_incidents"))
assert value is not None, data
assert int(value) >= 0, data
PY

curl -fsS -b "${COOKIE_JAR}" 'http://localhost:3000/api/proxy/query/queue?window=24h' | tee "${LOG_DIR}/web-queue.json" >/dev/null
python3 - "${LOG_DIR}/web-queue.json" <<'PY'
import json,sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    data=json.load(f)
assert isinstance(data, list), type(data)
PY

curl -fsS -b "${COOKIE_JAR}" 'http://localhost:3000/api/proxy/query/search-suggest?q=ap' | tee "${LOG_DIR}/web-search-suggest.json" >/dev/null
python3 - "${LOG_DIR}/web-search-suggest.json" <<'PY'
import json,sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    data=json.load(f)
items = data.get("items", data if isinstance(data, list) else [])
assert isinstance(items, list), data
PY

touch "${PROOF_DIR}/S6_data_warm.ok"

echo "=== /scores non-empty via web proxy ===" | log
curl -fsS -b "${COOKIE_JAR}" 'http://localhost:3000/api/proxy/query/scores?window=24h&maxPoints=50' | tee "${LOG_DIR}/web-scores.json" >/dev/null
python3 - "${LOG_DIR}/web-scores.json" <<'PY'
import json, sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)
series = data.get("series", [])
assert len(series) > 0, f"Expected non-empty score series, got {len(series)}. Response: {data}"
print(f"  scores OK: {len(series)} points, latest model={series[-1].get('model_version','?')}", file=sys.stderr)
PY

echo "=== /candles non-empty via web proxy ===" | log
curl -fsS -b "${COOKIE_JAR}" 'http://localhost:3000/api/proxy/query/candles?window=24h&maxPoints=50' | tee "${LOG_DIR}/web-candles.json" >/dev/null
python3 - "${LOG_DIR}/web-candles.json" <<'PY'
import json, sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)
series = data.get("series", [])
assert len(series) > 0, f"Expected non-empty candle series, got {len(series)}. Response: {data}"
print(f"  candles OK: {len(series)} candles", file=sys.stderr)
PY

echo "=== inference /metrics histogram present ===" | log
curl -fsS http://localhost:8090/metrics | tee "${LOG_DIR}/inference-metrics.txt" >/dev/null
python3 - "${LOG_DIR}/inference-metrics.txt" <<'PY'
import sys
text = open(sys.argv[1]).read()
assert "inference_scoring_latency_seconds" in text, "Missing inference_scoring_latency_seconds"
assert "inference_messages_produced" in text, "Missing inference_messages_produced"
print("  inference metrics OK", file=sys.stderr)
PY

echo "=== Prometheus up==1 ===" | log
python3 scripts/prom_up_check.py http://localhost:9090 --from-config prometheus/prometheus.yml \
  | tee "${LOG_DIR}/prom_up_check.txt"
touch "${PROOF_DIR}/S4_prom_up.ok"

echo "=== Grafana provision proof via HTTP API ===" | log
GF_USER="${GF_USER:-admin}"
GF_PASS="${GF_PASS:-sentinel}"
curl -fsS "http://${GF_USER}:${GF_PASS}@localhost:3001/api/health" | tee "${LOG_DIR}/grafana_health.json" >/dev/null
curl -fsS "http://${GF_USER}:${GF_PASS}@localhost:3001/api/search?type=dash-db" | tee "${LOG_DIR}/grafana_search.json" >/dev/null
python3 - "${LOG_DIR}/grafana_search.json" <<'PY'
import json,sys
d=json.load(open(sys.argv[1]))
assert isinstance(d,list) and len(d)>0, "no dashboards found"
PY
touch "${PROOF_DIR}/S5_grafana_provisioned.ok"

echo "PASS runtime-proof" | log
emit_status "PASS" "runtime proof complete"
