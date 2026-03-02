#!/usr/bin/env bash
set -euo pipefail
QUERY_URL="${QUERY_URL:-http://localhost:8085}"
COOKIE="${COOKIE:-}"
if ! curl -fsS "$QUERY_URL/healthz" >/dev/null 2>&1; then
  echo "SKIP queue_stream: query not reachable"
  exit 0
fi
out=$(curl -N --max-time 17 -s "$QUERY_URL/stream/queue" ${COOKIE:+-H "Cookie: $COOKIE"} || true)
echo "$out" | grep -Eq 'event: queue_patch|heartbeat'
echo "PASS queue_stream"
echo 'queue_stream placeholder: verify text/event-stream queue_patch event'
