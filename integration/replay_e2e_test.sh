#!/usr/bin/env bash
set -euo pipefail
GOV_URL="${GOV_URL:-http://localhost:8084}"
QUERY_URL="${QUERY_URL:-http://localhost:8085}"
COOKIE="${COOKIE:-}"
if ! curl -fsS "$GOV_URL/healthz" >/dev/null 2>&1; then
  echo "SKIP replay_e2e: governance not reachable"
  exit 0
fi
resp=$(curl -fsS -X POST "$GOV_URL/replay/start" -H 'Content-Type: application/json' ${COOKIE:+-H "Cookie: $COOKIE"} -d '{}')
id=$(echo "$resp" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
status=$(echo "$resp" | sed -n 's/.*"status"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
[ -n "$id" ] && [ "$status" = "queued" ]
sleep 1
curl -fsS "$QUERY_URL/replay/$id" ${COOKIE:+-H "Cookie: $COOKIE"} | grep -Eq '"status"'
echo "PASS replay_e2e $id"
echo 'replay_e2e placeholder: verify /replay/start returns queued then /replay/{id} transitions'
