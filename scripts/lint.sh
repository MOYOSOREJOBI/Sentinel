#!/usr/bin/env bash
set -euo pipefail
bad=0
if command -v gofmt >/dev/null 2>&1; then
  out=$(gofmt -l $(find cmd internal scripts -name '*.go' -not -path '*/vendor/*'))
  if [ -n "$out" ]; then
    echo "gofmt issues:"; echo "$out"; bad=1
  fi
fi
if command -v python3 >/dev/null 2>&1; then
  python3 -m py_compile services/inference/app.py
fi
if command -v node >/dev/null 2>&1 && [ -d web/node_modules ]; then
  (cd web && npm run -s lint:fix >/dev/null)
fi
if [ $bad -eq 1 ]; then exit 1; fi
echo "lint ok"
