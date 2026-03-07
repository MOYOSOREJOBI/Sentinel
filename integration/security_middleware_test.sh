#!/usr/bin/env bash
set -euo pipefail
GW_URL="${GW_URL:-http://localhost:8080}"
QUERY_URL="${QUERY_URL:-http://localhost:8085}"
WEB_URL="${WEB_URL:-http://localhost:3000}"
if ! curl -fsS "$GW_URL/healthz" >/dev/null 2>&1; then
  echo "SKIP security_middleware: gateway not reachable"
  exit 0
fi
# Unauthenticated logout must fail with auth semantics before clearing cookies.
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$GW_URL/auth/logout")
[ "$code" = "401" ]
COOKIE_JAR="$(mktemp /tmp/sentinel-security-cookie.XXXXXX)"
trap 'rm -f "$COOKIE_JAR"' EXIT
proxy_login=$(curl -sS -c "$COOKIE_JAR" -X POST "$WEB_URL/api/proxy/gateway-api/auth/login" -H 'Content-Type: application/json' -d '{"Email":"admin@sentinel.local","Password":"Sentinel#123"}')
printf '%s' "$proxy_login" | grep -q '"role"'
code=$(curl -s -o /dev/null -w '%{http_code}' -b "$COOKIE_JAR" "$WEB_URL/api/proxy/gateway-api/me")
[ "$code" = "200" ]

csrf=$(awk '$6=="sentinel_csrf"{print $7}' "$COOKIE_JAR" | tail -n 1)
[ -n "$csrf" ]

# Authenticated query writes must fail closed without a CSRF token.
if curl -fsS "$QUERY_URL/healthz" >/dev/null 2>&1; then
  code=$(curl -s -o /dev/null -w '%{http_code}' -b "$COOKIE_JAR" -X POST "$QUERY_URL/dev/backfill?years=1")
  [ "$code" = "403" ]
fi

# Viewer must remain blocked on admin write routes even with a valid CSRF token.
VIEWER_JAR="$(mktemp /tmp/sentinel-security-viewer.XXXXXX)"
trap 'rm -f "$COOKIE_JAR" "$VIEWER_JAR"' EXIT
curl -fsS -c "$VIEWER_JAR" -X POST "$WEB_URL/api/proxy/gateway-api/auth/login" -H 'Content-Type: application/json' -d '{"Email":"viewer@sentinel.local","Password":"Sentinel#123"}' >/dev/null
viewer_csrf=$(awk '$6=="sentinel_csrf"{print $7}' "$VIEWER_JAR" | tail -n 1)
[ -n "$viewer_csrf" ]
code=$(curl -s -o /dev/null -w '%{http_code}' -b "$VIEWER_JAR" -X POST "$WEB_URL/api/proxy/query/dev/backfill?years=1" -H "X-CSRF-Token: $viewer_csrf")
[ "$code" = "403" ]

# Repeated bad logins must throttle.
rate_codes=""
for i in 1 2 3 4 5 6; do
  rate_codes="$rate_codes $(curl -s -o /dev/null -w '%{http_code}' -X POST "$GW_URL/auth/login" -H 'Content-Type: application/json' -d '{"Email":"x","Password":"y"}' || true)"
done
printf '%s' "$rate_codes" | grep -q '429'
echo "PASS security_middleware"
