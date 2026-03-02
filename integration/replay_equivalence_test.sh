#!/usr/bin/env bash
set -euo pipefail

TS_DIR="${TS_DIR:-docs/audit/_latest}"
PROOF_DIR="${TS_DIR}/proofs"
LOG_DIR="${TS_DIR}/logs"
mkdir -p "${PROOF_DIR}" "${LOG_DIR}"

COOKIE_JAR="$(mktemp /tmp/sentinel-replay-cookie.XXXXXX)"
REPLAY_JSON="${LOG_DIR}/replay-start.json"
REPLAY_STATUS_JSON="${LOG_DIR}/replay-status.json"
AUDIT_VERIFY_JSON="${LOG_DIR}/audit-verify.json"
trap 'rm -f "${COOKIE_JAR}"' EXIT

curl -fsS -c "${COOKIE_JAR}" \
  -H 'content-type: application/json' \
  -X POST http://localhost:8080/auth/login \
  --data '{"Email":"admin@sentinel.local","Password":"Sentinel#123"}' \
  > "${LOG_DIR}/replay-login.json"

csrf_token="$(awk '$6=="sentinel_csrf"{print $7}' "${COOKIE_JAR}" | tail -n1)"
if [ -z "${csrf_token}" ]; then
  echo "missing csrf token" >&2
  exit 1
fi

curl -fsS -b "${COOKIE_JAR}" \
  -H 'content-type: application/json' \
  -H "X-CSRF-Token: ${csrf_token}" \
  -X POST http://localhost:8084/replay/start \
  --data "$(python3 - <<'PY'
from datetime import datetime, timedelta, timezone
import json
end = datetime.now(timezone.utc)
start = end - timedelta(seconds=30)
print(json.dumps({
    "replay_mode": "recompute",
    "start": start.isoformat().replace("+00:00", "Z"),
    "end": end.isoformat().replace("+00:00", "Z"),
}))
PY
)" \
  > "${REPLAY_JSON}"

job_id="$(python3 - "${REPLAY_JSON}" <<'PY'
import json,sys
with open(sys.argv[1], 'r', encoding='utf-8') as f:
    data = json.load(f)
job_id = str(data.get('id', '')).strip()
if not job_id:
    raise SystemExit(1)
print(job_id)
PY
)"

deadline=$((SECONDS+90))
while true; do
  curl -fsS -b "${COOKIE_JAR}" "http://localhost:8085/replay/${job_id}" > "${REPLAY_STATUS_JSON}"
  if python3 - "${REPLAY_STATUS_JSON}" <<'PY'
import json,sys
with open(sys.argv[1], 'r', encoding='utf-8') as f:
    data = json.load(f)
status = data.get('status')
result = data.get('result') or {}
if status != 'completed':
    raise SystemExit(1)
if not isinstance(result, dict) or not result:
    raise SystemExit(1)
if int(result.get('recomputed_count', 0)) < 0:
    raise SystemExit(1)
PY
  then
    break
  fi

  if [ "${SECONDS}" -gt "${deadline}" ]; then
    echo "replay job did not complete" >&2
    exit 1
  fi
  sleep 2
done

curl -fsS -b "${COOKIE_JAR}" http://localhost:8084/audit/verify > "${AUDIT_VERIFY_JSON}"
python3 - "${AUDIT_VERIFY_JSON}" <<'PY'
import json,sys
with open(sys.argv[1], 'r', encoding='utf-8') as f:
    data = json.load(f)
assert data.get('ok') is True, data
PY

touch "${PROOF_DIR}/S9_replay_parity.ok"
