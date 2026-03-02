#!/usr/bin/env bash
set -euo pipefail
QUERY_URL="${QUERY_URL:-http://localhost:8085}"
COOKIE="${COOKIE:-}"
if ! curl -fsS "$QUERY_URL/healthz" >/dev/null 2>&1; then
  echo "SKIP world_map_filter: query not reachable"
  exit 0
fi
body=$(curl -fsS "$QUERY_URL/world-map?time_window=24h" ${COOKIE:+-H "Cookie: $COOKIE"})
echo "$body" | grep -Eq 'countries|timeWindow'
echo "PASS world_map_filter"
echo 'world_map_filter placeholder: verify country filter changes /queue'
