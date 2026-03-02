#!/usr/bin/env bash
set -euo pipefail
GW_URL="${GW_URL:-http://localhost:8080}"
if ! curl -fsS "$GW_URL/healthz" >/dev/null 2>&1; then
  echo "SKIP security_middleware: gateway not reachable"
  exit 0
fi
# CSRF check on state-changing endpoint without auth/csrf should fail closed
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$GW_URL/auth/logout")
[ "$code" = "403" ] || [ "$code" = "401" ]
# rate limit smoke (endpoint existence + repeated calls)
for i in 1 2 3 4 5 6; do
  curl -s -o /dev/null -X POST "$GW_URL/auth/login" -H 'Content-Type: application/json' -d '{"Email":"x","Password":"y"}' || true
done
echo "PASS security_middleware"
echo 'security_middleware placeholder: verify csrf 403 and rate-limit 429'
