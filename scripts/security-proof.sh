#!/usr/bin/env bash
set -euo pipefail

gateway_url="${GATEWAY_URL:-http://localhost:8080}"
alerts_url="${ALERTS_URL:-http://localhost:8083}"
governance_url="${GOVERNANCE_URL:-http://localhost:8084}"
proof_ip="${PROOF_IP:-10.10.10.10}"

admin_cookie="$(mktemp)"
viewer_cookie="$(mktemp)"
trap 'rm -f "$admin_cookie" "$viewer_cookie"' EXIT

extract_csrf() {
  local jar="$1"
  awk '$6 == "sentinel_csrf" { print $7 }' "$jar" | tail -n 1
}

login() {
  local jar="$1"
  local email="$2"
  local password="$3"
  curl -sS -c "$jar" -H 'content-type: application/json' \
    -H "X-Forwarded-For: $proof_ip" \
    -X POST "$gateway_url/auth/login" \
    --data "{\"Email\":\"$email\",\"Password\":\"$password\"}" >/dev/null
}

status_for() {
  local expected="$1"
  local actual="$2"
  local label="$3"
  echo "$label=$actual"
  if [[ "$actual" != "$expected" ]]; then
    echo "FAIL: $label expected $expected got $actual" >&2
    exit 1
  fi
}

unauth_status="$(curl -i -sS -o /dev/null -w '%{http_code}' -H "X-Forwarded-For: $proof_ip" -X POST "$governance_url/replay/start")"
status_for "401" "$unauth_status" "unauth_replay_start"

login "$viewer_cookie" "viewer@sentinel.local" "Sentinel#123"
viewer_csrf="$(extract_csrf "$viewer_cookie")"
viewer_status="$(curl -i -sS -o /dev/null -w '%{http_code}' -b "$viewer_cookie" -H "X-CSRF-Token: $viewer_csrf" -H "X-Forwarded-For: $proof_ip" -X POST "$governance_url/replay/start")"
status_for "403" "$viewer_status" "viewer_replay_start"

login "$admin_cookie" "admin@sentinel.local" "Sentinel#123"
admin_csrf="$(extract_csrf "$admin_cookie")"
admin_status="$(curl -i -sS -o /dev/null -w '%{http_code}' -b "$admin_cookie" -H "X-CSRF-Token: $admin_csrf" -H "X-Forwarded-For: $proof_ip" -X POST "$governance_url/replay/start")"
status_for "202" "$admin_status" "admin_replay_start"

logout_status="$(curl -i -sS -o /dev/null -w '%{http_code}' -b "$admin_cookie" -H "X-Forwarded-For: $proof_ip" -X POST "$gateway_url/auth/logout")"
status_for "403" "$logout_status" "logout_without_csrf"

rate_limit_hit=0
for _ in $(seq 1 50); do
  code="$(curl -i -sS -o /dev/null -w '%{http_code}' -H 'content-type: application/json' \
    -H "X-Forwarded-For: $proof_ip" \
    -X POST "$gateway_url/auth/login" \
    --data '{"Email":"viewer@sentinel.local","Password":"wrong-password"}')"
  if [[ "$code" == "429" ]]; then
    rate_limit_hit=1
    echo "login_rate_limit=$code"
    break
  fi
done

if [[ "$rate_limit_hit" != "1" ]]; then
  echo "FAIL: login rate limit did not return 429 within 50 attempts" >&2
  exit 1
fi

echo "PASS: security proof"
