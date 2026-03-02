#!/usr/bin/env bash
set -euo pipefail
compose="docker compose -f deploy/docker/docker-compose.yml"
ready=0
for i in {1..90}; do
  if $compose exec -T redpanda rpk cluster info >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done
if [ "$ready" -eq 0 ]; then
  echo "ERROR: Redpanda failed to become ready after 90 seconds"
  exit 1
fi
echo "Redpanda is ready, creating topics..."
topics=(raw.ticks derived.candles derived.features derived.scores alerts.created dq.metrics dlq.aggregator dlq.features dlq.inference dlq.alerts)
for t in "${topics[@]}"; do
  $compose exec -T redpanda rpk topic create "$t" --partitions 3 --replicas 1 || true
done
for t in "${topics[@]}"; do
  $compose exec -T redpanda rpk topic describe "$t" >/dev/null
done
echo "Topics created and metadata available."
