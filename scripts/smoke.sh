#!/usr/bin/env bash
set -euo pipefail

base_api=${BASE_API:-http://localhost}
prom=${PROM_URL:-http://localhost:9090}

wait_http() {
  local url=$1
  local tries=${2:-60}
  for _ in $(seq 1 "$tries"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "timeout waiting for $url" >&2
  return 1
}

wait_http "$base_api:8080/readyz"
wait_http "$base_api:8085/readyz"
wait_http "$base_api:8083/readyz"
wait_http "$base_api:8084/readyz"
wait_http "$base_api:8090/readyz"

seed_ok=0
for _ in $(seq 1 60); do
  json=$(curl -fsS "$base_api:8085/debug/seed-status")
  state=$(echo "$json" | python3 -c 'import json,sys; d=json.load(sys.stdin); print("ok" if all(d.get(k,0)>0 for k in ["raw_ticks","candles","features","scores","alerts","incidents"]) else "wait")')
  if [[ "$state" == "ok" ]]; then
    seed_ok=1
    echo "seed-status ok: $json"
    break
  fi
  sleep 2
done
if [[ "$seed_ok" -ne 1 ]]; then
  echo "seed-status did not become non-zero" >&2
  exit 1
fi

up=$(curl -fsS "$prom/api/v1/query" --data-urlencode 'query=sum(up{job=~"gateway|aggregator|features|alerts|governance|query|inference"})' | python3 -c 'import json,sys; d=json.load(sys.stdin); r=d.get("data",{}).get("result",[]); print(r[0]["value"][1] if r else "0")')
if [[ "$up" != "7" ]]; then
  echo "unexpected up sum=$up (expected 7)" >&2
  exit 1
fi

echo "smoke checks passed"
