# Testing Guide

## Fast local checks
- `go test ./internal/rbac -count=1`
- `go test ./cmd/alerts -run TestIsAllowedCaseTransition -count=1`
- `go test ./cmd/query -run 'TestRatio|TestReplayLanes' -count=1`
- `cd web && npm test`

## Full unit layer
- `make unit`

## Integration suite (Docker-gated)
- `make integration-suite`
- Includes:
  - pipeline/idempotency/replay checks
  - dependency failure checks
  - case workflow regression (`integration/case_workflow_test.sh`)
  - query auth + governance RBAC regression (`integration/query_auth_test.sh`)

## Browser validation / artifacts
- Non-strict (returns `3` on environment SKIP):
  - `make browser-validate`
  - `make screenshot-smoke`
- Strict release mode (SKIP becomes failure):
  - `REQUIRE_BROWSER=1 make browser-validate`
  - `REQUIRE_BROWSER=1 make screenshot-smoke`
  - `make verify-screenshots`

## Release gate
- `make release-gate`
- Gate fails on:
  - unit / integration regressions
  - browser validation failure
  - screenshot failure
  - invalid or incomplete screenshot manifest

## Full Docker runtime validation
- Run from repo root.
- `bash scripts/final-validation.sh`
