# Detailed Audit Appendix

Timestamp basis: `20260303T091408Z` audit plus live runtime re-checks captured after booting the stack with a local port override.

## Scope

This appendix expands the original audit with:

- a codebase map of the active repo
- a stricter distinction between "fresh-boot reality" and "local override demo reality"
- a file-by-file summary of the core system
- a defect inventory focused on code paths that are broken, misleading, incomplete, or demo-only
- a production target vs current-state comparison

Primary source files:

- `README.md`
- `Makefile`
- `deploy/docker/docker-compose.yml`
- `deploy/docker/Dockerfile-go`
- `scripts/run-demo.sh`
- `scripts/doctor.sh`
- `scripts/playwright-docker.sh`
- `scripts/seed-users.go`
- `scripts/backfill.go`
- `cmd/gateway-api/main.go`
- `cmd/query/main.go`
- `cmd/alerts/main.go`
- `cmd/governance/main.go`
- `web/app/api/proxy/[service]/[...path]/route.ts`
- `web/lib/api.ts`
- `web/components/LoginScreen.tsx`
- `services/inference/app.py`

Primary evidence files:

- `docs/progress_check/20260303T091408Z/REPORT.md`
- `docs/progress_check/20260303T091408Z/SCORECARD.json`
- `docs/progress_check/20260303T091408Z/CLAIMS_VS_TRUTH.md`
- `docs/progress_check/20260303T091408Z/GAPS_AND_FIXES.md`
- `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_up.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_ps_live_override.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/curl/web_root_live.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/curl/gateway_healthz_live.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/curl/query_readyz_live.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_login_via_proxy.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_me_via_proxy.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_command_center_via_proxy.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_feed_via_proxy.txt`

## Two Different Realities

There are currently two materially different "truths" in this repo. They should not be conflated.

### 1. Fresh-Boot Audit Reality

This is the strict baseline used in the original evidence-based audit.

- `production_readiness_percent` is `25`.
- `demo_completion_percent` is `0`.
- The raw documented compose path failed before the app stack came up.
- Because the boot failed, most runtime proofs for DB, UI, metrics, security, and Playwright were either negative evidence or `UNKNOWN`.

Evidence:

- `docs/progress_check/20260303T091408Z/SCORECARD.json`
- `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_up.txt`

### 2. Local Override Demo Reality

This is the state reached after manually correcting local prerequisites and avoiding host port collisions.

- All core runtime containers are up and healthy.
- The web app is reachable on `:3000`.
- The gateway and query services are reachable on remapped ports `:58080` and `:58085`.
- Login through the real browser proxy path succeeds.
- Authenticated read endpoints return data.

Evidence:

- `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_ps_live_override.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/curl/web_root_live.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/curl/gateway_healthz_live.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/curl/query_readyz_live.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_login_via_proxy.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_me_via_proxy.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_command_center_via_proxy.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_feed_via_proxy.txt`

This second state proves "the demo can run". It does not replace the first state for production readiness scoring.

## Repo Map

### Core product code

- `cmd/`
  - Go service entrypoints.
  - Active services: `gateway-api`, `query`, `alerts`, `governance`, `aggregator`, `features`, `simulator`.
- `internal/`
  - Shared application libraries.
  - Key domains: auth, RBAC, middleware, metrics, replay, audit, query models, pipeline logic.
- `services/inference/`
  - Python inference service.
  - Owns scoring model loading, Kafka consumer loop, `/readyz`, and `/metrics`.
- `web/`
  - Next.js UI, API proxy layer, SSE proxy routes, browser tests.
- `sql/migrations/`
  - Database schema and lineage materialization.
- `deploy/docker/`
  - Local orchestration and Dockerfiles.

### Operational scripts

- `scripts/`
  - Bootstrapping, validation, smoke tests, browser scripts, Playwright wrapper, local proof scripts.
  - Also contains two Go CLIs that currently break package-wide `go test` and `go vet`.

### Integration and proof layers

- `integration/`
  - End-to-end and security-style shell tests.
- `docs/progress_check/...`
  - Historical audit outputs and evidence snapshots.

### Noise / generated artifacts present in the repo

- `web/.next/`
  - Generated Next build cache and dev output.
- `web/node_modules/`
  - Installed frontend dependencies.
- Historical evidence folders under `docs/`
  - Valuable for provenance, but they increase search noise during engineering work.

## Core File Summary

### `README.md`

Role:

- Public-facing operator and developer guide.

Current value:

- Correctly points users to `make demo` as the main bring-up path.
- Correctly documents compose-mode upstream env behavior for the web proxy.

Current problems:

- The manual compose fallback is incomplete and false on a clean checkout.
- It implies a simpler boot path than the code actually requires.

Key lines:

- `README.md:66-76`
- `README.md:138-143`
- `README.md:147-168`

### `Makefile`

Role:

- Main developer entrypoint.

Current value:

- Centralizes demo, test, validation, and integration commands.
- `make demo` points to the safer boot script.

Current problems:

- `up` does not generate keys before build.
- `seed` assumes Postgres on `localhost:5432`.
- `test` and `unit` use `./...`, which currently includes a broken `scripts` package.

Key lines:

- `Makefile:17-21`
- `Makefile:35-36`
- `Makefile:56-62`
- `Makefile:67-68`

### `deploy/docker/docker-compose.yml`

Role:

- Canonical local runtime topology.

Current value:

- Clearly defines the intended architecture and dependencies.
- Correctly wires compose-internal DNS for the web layer.
- Includes optional Playwright service and monitoring services.

Current problems:

- Publishes fixed host ports for almost every dependency and service.
- This makes it fragile on developer machines where common ports are already occupied.
- The base file doubles as both developer stack and E2E stack, which is operationally brittle.

Key lines:

- `deploy/docker/docker-compose.yml:1-270`

### `deploy/docker/Dockerfile-go`

Role:

- Shared image build for all Go services.

Current value:

- Single build path for all Go services.

Current problems:

- Hard requirement on `COPY keys ./keys` means image build fails if repo-root keys are absent.
- This makes the raw compose path fail before containers start.

Key lines:

- `deploy/docker/Dockerfile-go:9-14`

### `scripts/run-demo.sh`

Role:

- The only truly workable repo-level bring-up path today.

Current value:

- Generates keys first.
- Starts infra first.
- Runs migrations.
- Creates Kafka topics.
- Starts app services.
- Seeds demo users.
- Performs lightweight readiness checks and basic query-path validation.

Current problems:

- Assumes the default host ports are free.
- Still uses host-bound URLs for validation.
- It is a developer/demo boot script, not a production orchestration path.

Key lines:

- `scripts/run-demo.sh:23-49`

### `scripts/doctor.sh`

Role:

- Local preflight check.

Current value:

- Detects missing commands, Docker availability, and occupied ports.

Current problems:

- Missing JWT key files are logged, but do not set `warn=1`.
- This lets `doctor` understate a broken boot precondition.

Key lines:

- `scripts/doctor.sh:17-30`

### `scripts/playwright-docker.sh`

Role:

- Wrapper around the Playwright compose service.

Current value:

- Minimal entrypoint for E2E browser execution.

Current problems:

- No port preflight.
- No E2E-specific compose layering.
- Writes success marker unconditionally after the compose command returns successfully, but depends entirely on the base compose file.

Key lines:

- `scripts/playwright-docker.sh:3-9`

### `scripts/seed-users.go`

Role:

- Seeds demo users, instruments, watchlists, and preferences.

Current value:

- Required for login and useful seeded UI state in local demos.

Current problems:

- Lives in `scripts/` as `package main`.
- Shares that directory with `scripts/backfill.go`, creating a duplicate `main` package under `./...`.
- Defaults to `localhost:5432`, which is wrong whenever the stack is run via a host-port override.
- The seeded data is intentionally demo-grade and thin.

Key lines:

- `scripts/seed-users.go:26-30`
- `scripts/seed-users.go:38-45`
- `scripts/seed-users.go:47-105`

### `scripts/backfill.go`

Role:

- Manual CLI for historical query backfill.

Current value:

- Provides a direct operator/developer mechanism for loading historical data.

Current problems:

- Same package-layout issue as `scripts/seed-users.go`.
- Defaults to `localhost:5432`.
- Exposed in the UI through a dev path, which is not a production-safe operator model.

Key lines:

- `scripts/backfill.go:15-43`

### `cmd/gateway-api/main.go`

Role:

- Auth entrypoint.
- Issues session and CSRF cookies.
- Owns login, logout, and `/me`.

Current value:

- Has login rate limiting.
- Uses CSRF middleware globally.
- Properly returns `401` for unauthenticated `/me`.

Current problems:

- `POST /auth/logout` does not require an authenticated subject to execute.
- It still clears cookies and returns `204` even if no valid session exists.
- That is not catastrophic, but it is a real auth semantics gap for a write route.

Key lines:

- `cmd/gateway-api/main.go:48-58`
- `cmd/gateway-api/main.go:81-115`
- `cmd/gateway-api/main.go:117-125`
- `cmd/gateway-api/main.go:128-135`

### `cmd/query/main.go`

Role:

- Read-model service for the UI.
- Also exposes a dev backfill action.

Current value:

- Healthy read surface for queue, command center, feed, trust, scores, and supporting data.
- Runtime-proven via local override demo.

Current problems:

- No router-level CSRF middleware at all.
- `POST /dev/backfill` is admin-gated, but not CSRF-protected.
- This is the cleanest currently provable live security gap in the repo.

Key lines:

- `cmd/query/main.go:104-148`
- `cmd/query/main.go:150+`

### `cmd/alerts/main.go`

Role:

- Consumes `derived.scores`.
- Persists scores and alerts.
- Exposes alert and incident mutations.

Current value:

- Uses CSRF middleware.
- Uses RBAC and rate limiting for write routes.
- Is architecturally closer to production than the boot/test tooling around it.

Current problems:

- Runtime proof of deep persistence and long-lived correctness is still thinner than desired.
- Current live demo data remains shallow, so durability is not yet proven under sustained load.

Key lines:

- `cmd/alerts/main.go:77-119`
- `cmd/alerts/main.go:421-499`

### `cmd/governance/main.go`

Role:

- Model deployment, replay jobs, escalation policies, audit verification.

Current value:

- Uses CSRF middleware.
- Uses separate RBAC wrappers for reads, model deploy, workflow, and policy admin.

Current problems:

- Runtime behavior is not fully proven in the current live checks.
- The route surface is broader than the currently exercised proof coverage.

Key lines:

- `cmd/governance/main.go:59-81`
- `cmd/governance/main.go:116-260`

### `web/app/api/proxy/[service]/[...path]/route.ts`

Role:

- Server-side proxy for all browser API requests.

Current value:

- Correctly switches between compose DNS mode and local host mode.
- Transparently forwards cookies and response headers.

Current problems:

- It is a blind passthrough for `GET`, `POST`, `PUT`, `PATCH`, and `DELETE`.
- It does not enforce any local auth or CSRF policy itself.
- That is acceptable only if upstream services are consistently correct, which they currently are not for every write path.

Key lines:

- `web/app/api/proxy/[service]/[...path]/route.ts:11-39`
- `web/app/api/proxy/[service]/[...path]/route.ts:44-48`

### `web/lib/api.ts`

Role:

- Client-side API wrapper.

Current value:

- Attaches `X-CSRF-Token` from the browser cookie on non-GET requests.
- Provides a consistent frontend integration layer.

Current problems:

- It hides backend failures behind broad `.catch(...)` fallbacks on many read routes.
- This makes the UI more demo-friendly, but it can mask backend breakage and make the product look healthier than it is.
- It exposes the dev backfill route directly to the UI client.

Key lines:

- `web/lib/api.ts:9-15`
- `web/lib/api.ts:29-30`
- `web/lib/api.ts:31-45`
- `web/lib/api.ts:83-90`

### `web/components/LoginScreen.tsx`

Role:

- Primary login screen.

Current value:

- Uses the real login flow and verifies `/me` after login.

Current problems:

- Defaults to demo credentials in the form fields.
- This is acceptable for a demo, but not production UX.
- It reveals that the product assumes a seeded local demo operator account.

Key lines:

- `web/components/LoginScreen.tsx:9-10`
- `web/components/LoginScreen.tsx:14-20`

### `services/inference/app.py`

Role:

- Inference scoring service.

Current value:

- Exposes `/healthz`, `/readyz`, and `/metrics`.
- Returns lineage-like readiness fields including `fallback_mode`, `model_version`, and `model_artifact_hash`.

Current problems:

- The service is live in the current override stack, but this appendix does not add full end-to-end scoring validation beyond the existing feed output.
- Production confidence still depends on stronger pipeline and persistence proofs than a single live feed sample.

Key lines:

- `services/inference/app.py:246-276`

## Runtime-Proven Working Paths Right Now

These are the things that are no longer hypothetical in the current local override stack.

- All major app containers are healthy in the live override stack.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_ps_live_override.txt`
- The web root responds with `200`.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/curl/web_root_live.txt`
- `gateway-api` `/healthz` responds with `200 ok`.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/curl/gateway_healthz_live.txt`
- `query` `/readyz` responds with `200 ok`.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/curl/query_readyz_live.txt`
- Browser-path login via Next proxy succeeds and sets both auth and CSRF cookies.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_login_via_proxy.txt`
- Authenticated identity lookup works.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_me_via_proxy.txt`
- Authenticated command-center read works.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_command_center_via_proxy.txt`
- Authenticated feed read works.
  - Proof: `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_feed_via_proxy.txt`

This proves a real local demo path exists.

## Broken, Misleading, Or Incomplete Code Paths

### 1. Broken boot path in the documented manual flow

Problem:

- `README.md` tells the user they can directly run raw compose.
- The shared Go Docker image requires repo-root keys.
- `make up` does not generate those keys.

Broken chain:

- `README.md:72-76`
- `Makefile:20-21`
- `deploy/docker/Dockerfile-go:13`

Effect:

- A clean checkout can fail before any app container starts.

Why this matters:

- This is not just a doc bug. It breaks reproducibility and invalidates the fastest advertised bring-up path.

### 2. `doctor` can understate a broken local environment

Problem:

- Missing JWT keys print warnings but do not set the warning flag.

Code path:

- `scripts/doctor.sh:25-26`

Effect:

- A developer can see key warnings and still get a softer overall result than the boot path deserves.

Why this matters:

- It weakens the only lightweight local preflight check.

### 3. Package-wide Go test and vet are structurally broken

Problem:

- `scripts/seed-users.go` and `scripts/backfill.go` both define `main()` in the same directory.

Code path:

- `scripts/seed-users.go:26`
- `scripts/backfill.go:15`
- `Makefile:56-62`

Effect:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`

all fail when they include `sentinel/scripts` as a package.

Why this matters:

- You do not have a trustworthy repo-wide Go quality gate.

### 4. Real CSRF gap in `query`

Problem:

- `query` does not install `RequireCSRFFunc`.
- It exposes an admin write route.

Code path:

- `cmd/query/main.go:104-148`

Effect:

- `POST /dev/backfill` is admin-protected but not CSRF-protected.

Why this matters:

- This is a live security inconsistency between services.

### 5. `logout` semantics are weaker than they should be

Problem:

- `POST /auth/logout` is not wrapped in explicit auth.

Code path:

- `cmd/gateway-api/main.go:117-125`

Effect:

- The route still mutates cookies and returns success with no valid session.

Why this matters:

- In production, write routes should have unambiguous auth semantics.

### 6. The web proxy is intentionally permissive

Problem:

- The Next proxy forwards all verbs with the request headers/body as-is.

Code path:

- `web/app/api/proxy/[service]/[...path]/route.ts:26-39`
- `web/app/api/proxy/[service]/[...path]/route.ts:44-48`

Effect:

- Browser-side safety depends entirely on upstream correctness.

Why this matters:

- This is fine only if every upstream write route is correct.
- The repo already contains at least one upstream write inconsistency (`query`).

### 7. The UI intentionally hides some backend failures

Problem:

- Many read APIs catch errors and return placeholder defaults.

Code path:

- `web/lib/api.ts:31-45`
- `web/lib/api.ts:57-89`

Effect:

- The UI can still render and appear "mostly fine" while backend read paths are broken.

Why this matters:

- This is useful for demo resilience.
- It is dangerous for production diagnostics because it can suppress symptoms.

### 8. The login screen is seeded for demos, not real operator onboarding

Problem:

- The login form ships with pre-filled demo credentials.

Code path:

- `web/components/LoginScreen.tsx:9-10`

Effect:

- The product assumes seeded local users.

Why this matters:

- This is not acceptable as-is for production authentication UX.

### 9. Playwright path is operationally under-engineered

Problem:

- It reuses the base compose file.
- It does not preflight conflicting ports.

Code path:

- `scripts/playwright-docker.sh:3-8`
- `deploy/docker/docker-compose.yml` shared host port bindings

Effect:

- E2E can fail for environmental reasons unrelated to the app itself.

Why this matters:

- A flaky E2E harness is not a production gate.

### 10. The live demo data is thin and synthetic

Observed runtime state:

- `command-center` showed `openIncidents: 4`
- `topCountries` was `XX`
- `topIndustries` was `Unknown`
- `feed` had only one day of history
- feed rows contained empty `region`, `sector`, `venue`, and `industry`

Proof:

- `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_command_center_via_proxy.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/ui/live_feed_via_proxy.txt`

Why this matters:

- This is enough to prove the UI path works.
- This is not enough to claim production-grade data quality or model realism.

## What Is Demo-Only Today

These behaviors are acceptable for an internal demo, but not enough for production.

- Demo credentials are the default browser login path.
- User accounts must be manually seeded for the demo to be usable after raw boot.
- Thin seeded watchlists and preferences create the appearance of personalized views.
- The UI’s fallback responses hide missing data or failed reads in multiple places.
- The dev backfill path is directly exposed in the client API layer.
- The live dataset observed in this run is small and obviously synthetic-looking.

## Current Limitations By Subsystem

### Backend APIs

Now:

- Main services exist and can respond.
- Core read paths are functional in the live override stack.

Still missing for production:

- Deterministic, documented boot under the default supported path.
- Full live route-by-route proof under the canonical stack.
- Removal of dev-only write surfaces from the production route set.

### Data pipeline

Now:

- Data is flowing enough to generate incidents and feed rows.

Still missing for production:

- Sustained end-to-end proof over longer windows.
- Stronger evidence of replay parity, data durability, and recoverability.
- Better data completeness and non-placeholder metadata in incident output.

### Database

Now:

- The local override stack includes a healthy Postgres container.

Still missing for production:

- A refreshed post-fix SQL truth run that proves row counts, joins, lineage fields, and constraints under the current live boot.
- Formal proof that the live DB state stays correct across restarts and replay operations.

### ML / Quant

Now:

- Inference exposes readiness and metrics surfaces.
- Feed rows include `modelVersion`.

Still missing for production:

- Strong runtime proof of model artifact management, fallback correctness, and drift monitoring.
- Non-demo data coverage and realistic historical depth.

### UI / UX

Now:

- The browser app works and can render authenticated views.

Still missing for production:

- Removal of demo defaults.
- Better failure transparency.
- Better alignment between visible UI state and backend truth.

### Security

Now:

- Auth, RBAC, CSRF, and rate limiting exist in meaningful parts of the stack.

Still missing for production:

- Eliminate route inconsistencies (`query` CSRF, logout auth semantics).
- Re-run full runtime auth/RBAC/CSRF proof against the fixed stack.
- Confirm no other forgotten write routes exist outside the already-audited set.

### Observability

Now:

- Metrics endpoints exist in code.

Still missing for production:

- Reproducible boot with Prometheus and Grafana in the active local stack.
- Current-run proof that scrape targets, dashboards, and useful metrics are actually present.

### Testing

Now:

- There are unit tests and shell integration tests in the repo.

Still missing for production:

- A passing package-wide Go test/vet/race gate.
- A stable E2E browser gate.
- CI paths that exercise the real supported boot flow instead of a lucky local environment.

### Production operations

Now:

- The repo has useful scripts, but they encode operator tribal knowledge.

Still missing for production:

- A single canonical boot path.
- A single canonical validation path.
- Cleaner separation between local demo convenience and production-safe operations.

## Production Target Vs Current State

### What production should look like

- One canonical bring-up path works on a clean checkout without hidden preconditions.
- All test gates pass in CI without manual exclusions or local heroics.
- All write routes have consistent auth, RBAC, CSRF, and rate limiting.
- Demo-only routes are disabled or removed from production builds.
- Observability is live, scraped, and actionable.
- UI failures surface truthfully rather than silently defaulting.
- Data is deep enough, realistic enough, and proven durable enough for operator trust.
- Documentation matches reality and is kept current.

### Where the repo is now

- A demo can be made to run locally.
- The advertised manual boot path is still broken on a clean repo.
- Security is substantial but inconsistent.
- Test infrastructure is present but not trustworthy as a gate.
- Monitoring exists on paper but remains insufficiently proven in the current baseline.
- The UI is functional, but its defaults and fallback behavior are still optimized for a demo.
- Data exists, but the currently proven data slice is too small and too synthetic for production trust.

## Fastest Path To Production

This is the shortest dependency-ordered sequence that closes the most important gaps first.

### Stage 1: Fix reproducibility

- Make `make up` generate keys before building.
- Correct README so the documented manual path is actually truthful.
- Add compose overrides so local and E2E boots do not depend on uncontested default host ports.

Why first:

- Every other proof depends on a stable, reproducible boot path.

### Stage 2: Restore trustworthy engineering gates

- Remove `sentinel/scripts` from package-wide test/vet targets or move each Go CLI into its own subdirectory.
- Get `go test ./...`, `go test -race ./...`, and `go vet ./...` green.

Why second:

- Without this, every further change increases uncertainty.

### Stage 3: Close the proven security gaps

- Add CSRF middleware to `query`.
- Require auth on `POST /auth/logout`.

Why third:

- These are small fixes with high security leverage.

### Stage 4: Re-run runtime truth under the fixed canonical boot

- Re-run DB truth.
- Re-run metrics truth.
- Re-run unauth, viewer, CSRF, and rate-limit probes.
- Re-run the browser and Playwright proof paths.

Why fourth:

- This converts the repo from "code looks right" to "system is proven right".

### Stage 5: Harden beyond demo mode

- Remove or gate dev-only routes in production.
- Remove prefilled demo credentials from the login UX.
- Reduce UI fallback masking for important operational screens.
- Deepen the seeded or real data path so operators are not looking at placeholder fields.

Why fifth:

- This is the point where the product stops merely presenting well and starts behaving like a reliable operator tool.

## Unknowns That Still Need Proof

These are not assumptions. They remain unproven in the current evidence set.

- Full Prometheus scrape and Grafana dashboard truth under the corrected runtime.
- Full SQL lineage truth after the now-running local override boot.
- End-to-end replay parity in the current live stack.
- Whether any other write route outside the already-audited surfaces is missing a middleware wrapper.
- Whether the current model outputs remain stable and sensible over longer-lived ingest windows.

## Bottom Line

The repo is best described as:

- a functioning local demo stack
- with meaningful real backend code
- with partially real security controls
- with a legitimate end-to-end path
- but still with demo-biased UX and operational shortcuts
- and with several critical engineering and operational gaps that prevent calling it production-ready

It is not a fake project. It is also not ready to ship as a production system without first fixing boot reproducibility, test integrity, security consistency, and runtime proof coverage.
