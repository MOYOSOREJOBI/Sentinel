# Sentinel Deep Diagnostic Audit

Timestamp: `20260303T091408Z`

Repo snapshot:

- HEAD: `908dc9d797bc837050ad715b4fb735d8e034f92a`
- File count: `681`
- Working tree before this audit was clean except for the new audit folder.
- Evidence:
  - `docs/progress_check/20260303T091408Z/EVIDENCE/repo/repo_state.txt`
  - `docs/progress_check/20260303T091408Z/INVENTORY/git_file_count.txt`

## Executive summary

- The biggest current blocker is not an application bug. A fresh raw compose boot fails before any services start because the Go Docker image requires a missing repo-root `keys/` directory.
- The documented manual compose quickstart is therefore false on a clean checkout.
- `make demo` is materially safer than raw compose because it generates dev keys, runs migrations, creates topics, and seeds users before demo validation.
- Static code shows that `alerts` now persists `scores` with lineage fields and that `governance` and `alerts` write routes are auth + RBAC gated.
- The older production-readiness review in `docs/PRODUCTION_READINESS_REVIEW.md` is stale in several important ways and now contradicts current code.
- `query` still has a real security gap: `POST /dev/backfill` is admin-gated but lacks a CSRF middleware.
- `gateway-api` logout is still a public write path because it does not require auth when no valid session is present.
- Package-wide Go test and vet are currently broken by the `sentinel/scripts` directory containing multiple `main` files as one package.
- Current runtime truth for DB, UI, metrics, and security is mostly negative evidence because the stack never reached a running state in the fresh-boot path.
- The Playwright Docker path exposed an additional operational issue: it can fail on host port `5432` conflicts even when the main stack is not up.

## What works now

Static code, not runtime:

- Service route wiring exists for all named services, including health, readiness, and metrics surfaces across the Go services and inference.
  - Evidence: `cmd/gateway-api/main.go:48-135`, `cmd/query/main.go:104-628`, `cmd/alerts/main.go:77-390`, `cmd/governance/main.go:59-342`, `cmd/aggregator/main.go:63-87`, `cmd/features/main.go:100-124`, `cmd/simulator/main.go:61-77`, `services/inference/app.py:246-276`
- `alerts` statically writes persisted `scores` with lineage fields, then creates `alerts` linked to `score_id`.
  - Evidence: `cmd/alerts/main.go:421-499`
- `run-demo.sh` uses a safer order than the stale internal review claims.
  - Evidence: `scripts/run-demo.sh:23-35`
- The web proxy is correctly configured to use service DNS in compose mode and localhost only in explicit local mode.
  - Evidence: `deploy/docker/docker-compose.yml:209-223`; `web/app/api/proxy/[service]/[...path]/route.ts:11-23`

## What fails or is misleading

- Fresh raw compose boot fails on missing `keys/`.
  - Proof: `deploy/docker/Dockerfile-go:13`; `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_up.txt`
- After that failed boot, every host curl probe for `gateway-api`, `query`, `alerts`, `governance`, `aggregator`, `features`, `simulator`, `inference`, `web`, `prometheus`, and `grafana` failed at connection time.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/curl/*`
- Database truth is unavailable because `postgres` never ran in the fresh-boot path.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/sql/tables.txt`
- UI truth is unavailable because `web` never ran in the fresh-boot path.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/ui/login_raw.txt`
- Security runtime truth is mostly unavailable because the target services were unreachable.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/security/*`
- The Playwright Docker path failed independently because host port `5432` was already allocated.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/playwright/results.txt`; `docs/progress_check/20260303T091408Z/EVIDENCE/playwright/port_5432_listener.txt`
- The README claims a raw compose path that does not currently work.
  - Proof: `README.md:72-76`; `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_up.txt`
- `docs/PRODUCTION_READINESS_REVIEW.md` still describes auth and metrics gaps that current source has already addressed, so it is no longer a trustworthy status source.
  - Proof: `docs/PRODUCTION_READINESS_REVIEW.md:38-65`; `docs/PRODUCTION_READINESS_REVIEW.md:115-145`; `cmd/alerts/main.go:116-119`; `cmd/governance/main.go:79-81`; `services/inference/app.py:274-276`

## Top 15 blockers

1. Fresh raw compose boot fails on missing `keys/`; fix the boot path to generate keys before build. Verify with `make up`.
2. `scripts/doctor.sh` does not count missing keys in its warning state; fix the warning accounting. Verify with `make doctor` on a repo without `keys/`.
3. README raw compose instructions are incomplete and currently false; fix the docs to match the real preconditions. Verify by following the updated steps literally.
4. `go test ./...` and `go vet ./...` are broken by `sentinel/scripts`; separate or exclude that package from package-wide test targets. Verify with `make test` and `go vet`.
5. `query` exposes `POST /dev/backfill` without CSRF; add router-level CSRF protection. Verify with admin requests with and without `X-CSRF-Token`.
6. `gateway-api` logout is not explicitly authenticated; require auth before logout. Verify with unauthenticated and authenticated logout probes.
7. The browser path cannot be validated while `web` is down; fixing the fresh boot path is a prerequisite for every UI claim. Verify with `POST /api/proxy/gateway-api/auth/login`.
8. Runtime DB truth is blocked until `postgres` starts; fixing the boot path is a prerequisite for every data claim. Verify with `docker compose ps` and `psql` inside `postgres`.
9. Runtime metrics truth is blocked until Prometheus and Grafana start; fix boot first, then re-run metrics probes. Verify with the Prometheus and Grafana curl commands captured in this audit.
10. Security runtime truth is blocked until the services start; fix boot first, then rerun unauth and viewer mutation probes.
11. The Playwright Docker path fails on host port `5432` conflicts; add preflight checks and/or isolate E2E compose networking. Verify with `./scripts/playwright-docker.sh`.
12. There is no E2E-specific compose override; add one so Playwright does not require the same host ports as normal local development. Verify with `docker compose config`.
13. The Playwright script is wired to a single compose file; teach it to use the E2E override automatically. Verify with script output and merged compose config.
14. The old production-readiness review is now stale enough to mislead debugging and prioritization; update it or stop treating it as current status. Verify by re-running this audit and reconciling every claim.
15. Historical audit output under `docs/` is large enough to add search noise and misdirection during debugging; archive or de-emphasize old evidence runs. Verify with a size check and a reduced active search surface.

## Next 72 hours plan

1. Make the local boot path deterministic: generate keys in `make up`, fix README, and re-run the full fresh-boot audit path.
2. Fix the `sentinel/scripts` package issue so package-wide Go test and vet become usable again.
3. Close the two live security gaps already proven statically: add CSRF to `query`, require auth on logout.
4. Stabilize the Playwright path by adding port preflight and an E2E compose override.
5. Re-run phases 3 through 9 after those changes and replace today’s negative runtime evidence with real service proofs.

## Next 2 weeks plan

1. Retire or rewrite stale readiness documents so status claims are derived from current evidence, not prior assumptions.
2. Normalize all operator scripts onto one reproducible boot strategy and one validation strategy.
3. Add CI gates for: fresh boot, package-wide Go test/vet, route-security expectations, and at least one end-to-end login plus query proof.
4. Reduce repo noise by archiving historical evidence runs and clearly separating reference-only folders from active source.
5. Re-run a full evidence audit after the boot path is fixed and use that run as the new baseline scorecard.

## Cross references

- `docs/progress_check/20260303T091408Z/SCORECARD.json`
- `docs/progress_check/20260303T091408Z/CLAIMS_VS_TRUTH.md`
- `docs/progress_check/20260303T091408Z/GAPS_AND_FIXES.md`
- `docs/progress_check/20260303T091408Z/ROUTE_SECURITY_MATRIX.md`
- `docs/progress_check/20260303T091408Z/DATA_TRUTH.md`
- `docs/progress_check/20260303T091408Z/USEFUL_VS_NOISE.md`
