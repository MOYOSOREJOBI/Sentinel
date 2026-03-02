#!/usr/bin/env bash
set -euo pipefail
STRICT=${REQUIRE_BROWSER:-0}
if ! command -v node >/dev/null 2>&1; then
  echo "SKIP: node unavailable"
  [ "$STRICT" = "1" ] && exit 1 || exit 3
fi
if node -e "require('playwright')" >/dev/null 2>&1; then
  node scripts/browser-validate.mjs
  exit $?
fi
if command -v docker >/dev/null 2>&1; then
  docker run --rm -v "$PWD":/work -w /work -e BASE_URL="${BASE_URL:-http://host.docker.internal:3000}" mcr.microsoft.com/playwright:v1.55.0-jammy node scripts/browser-validate.mjs
  exit $?
fi
echo "SKIP: playwright missing locally and docker unavailable"
[ "$STRICT" = "1" ] && exit 1 || exit 3
