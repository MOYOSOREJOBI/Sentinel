#!/usr/bin/env bash
set -euo pipefail
warn=0
for c in make openssl python3; do
  if ! command -v "$c" >/dev/null 2>&1; then
    echo "WARN missing: $c"
    warn=1
  fi
done
if ! command -v docker >/dev/null 2>&1; then
  echo "WARN missing: docker (required for demo/integration)"
  warn=1
elif ! docker info >/dev/null 2>&1; then
  echo "WARN docker daemon not reachable"
  warn=1
fi
if command -v ss >/dev/null 2>&1; then
  for p in 3000 3001 5432 6379 8080 8081 8082 8083 8084 8085 8090 9090 9092 11211; do
    if ss -ltn "sport = :$p" | grep -q LISTEN; then
      echo "WARN port in use: $p"
      warn=1
    fi
  done
fi
[ -f keys/jwtRS256.key ] || echo "WARN jwt private key missing; run make dev-keys"
[ -f keys/jwtRS256.key.pub ] || echo "WARN jwt public key missing; run make dev-keys"
if [ $warn -eq 0 ]; then
  echo "doctor ok"
else
  echo "doctor completed with warnings"
fi
