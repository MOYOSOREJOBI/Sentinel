#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-$ROOT/docs/routes_inventory.txt}"
mkdir -p "$(dirname "$OUT")"

{
  echo "# Route Inventory"
  echo
  echo "## Alerts"
  rg -n '^\s*r\.(Get|Post|Patch|Delete)|^\s*(read|mutation|workflow|admin|policyAdmin)\.(Get|Post|Patch|Delete)' "$ROOT/cmd/alerts/main.go"
  echo
  echo "## Governance"
  rg -n '^\s*r\.(Get|Post|Patch|Delete)|^\s*(read|workflow|admin|policyAdmin)\.(Get|Post|Patch|Delete)' "$ROOT/cmd/governance/main.go"
  echo
  echo "## Query"
  rg -n '^\s*r\.(Get|Post|Patch|Delete)|^\s*pr\.(Get|Post|Patch|Delete)' "$ROOT/cmd/query/main.go"
  echo
  echo "## Gateway"
  rg -n '^\s*r\.(Get|Post|Patch|Delete)' "$ROOT/cmd/gateway-api/main.go"
} > "$OUT"

echo "$OUT"
