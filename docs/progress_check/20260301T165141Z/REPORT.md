# Sentinel Progress Check + Truth Audit

Audit timestamp: `20260301T165141Z`

Scope: current working tree at `cd4310ec3dc498f6272e798696b6604e429a5586`, not a clean commit. Evidence root: `docs/progress_check/20260301T165141Z/`.

External `/mnt/data` inputs were unavailable in this environment (`docs/progress_check/20260301T165141Z/EVIDENCE/input_availability.txt`). Any claim requiring those files is `UNKNOWN`.

## Executive Summary

- The stack is runnable locally: `docker compose up -d --build` completed and the core services are healthy (`docs/progress_check/20260301T165141Z/EVIDENCE/docker/compose_ps.txt`).
- The live data pipeline is real, not stubbed: `raw_ticks=191527`, `candles=145160`, `features=191526`, `scores=191524` (`docs/progress_check/20260301T165141Z/EVIDENCE/sql/counts.txt`).
- `scores` are actually persisted by `alerts`, not merely claimed in docs: `INSERT INTO scores ... ON CONFLICT ... RETURNING id` exists at `cmd/alerts/main.go:467-470` and replay also writes scores at `internal/replay/runner.go:70` (`docs/progress_check/20260301T165141Z/EVIDENCE/sql/rg_scores_insert.txt`).
- The ML path is materially real: inference is not in fallback mode and exposes lineage/calibration metadata (`docs/progress_check/20260301T165141Z/EVIDENCE/metrics/inference_readyz.json`; `services/inference/app.py:183-243`, `services/inference/app.py:256-270`).
- Go-service observability is materially improved: HTTP, DB, and Kafka metrics exist for most services, and Prometheus/Grafana are up (`docs/progress_check/20260301T165141Z/EVIDENCE/metrics/go_metrics_summary.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/metrics/prom_up.json`, `docs/progress_check/20260301T165141Z/EVIDENCE/metrics/grafana_search.json`).
- Security is mixed: unauthenticated mutations are blocked and viewer RBAC is enforced, but CSRF was not enforced on logout and rate limiting did not trigger in the probe (`docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/alerts_ack_unauth.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/alerts_ack_viewer.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/gateway_logout_no_csrf.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/login_rate_limit_codes.txt`).
- The largest end-user failure is the current web-proxy auth path: `POST /api/proxy/gateway-api/auth/login` returned `401`, which cascades into degraded SSE and blocked UI data fetches (`docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_proxy_login_raw.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/curl/sse_queue.txt`).
- DNS inside the `web` container is correct, but `localhost` is not: `query:8085` works and `localhost:8085` fails from inside `web` (`docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_container_query_dns_wget.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_container_query_localhost_wget.txt`).
- Testing is the weakest production signal: Go statement coverage is `17.5%`, many critical entrypoints are `0.0%`, and Playwright login failed repeatedly (`docs/progress_check/20260301T165141Z/EVIDENCE/tests/cover_func.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt`).
- Demo gates are greener than reality: latest audit says `COMPLETION: 100%`, but its traceability file still contains stale skip notes, so audit PASS cannot be treated as sufficient proof by itself (`docs/audit/20260301T163152Z/RESULT.md`, `docs/audit/20260301T163152Z/TRACEABILITY.md`).

## What's True vs What's Fake

| Area | Claim | Truth | Evidence |
| --- | --- | --- | --- |
| Pipeline writes scores | “ticks -> features -> scores -> alerts” | True in runtime. `scores` is non-zero and `alerts` upserts score rows before creating alerts | `docs/progress_check/20260301T165141Z/EVIDENCE/sql/counts.txt`; `cmd/alerts/main.go:467-482` |
| Every service exposes `/metrics` | README says every service exposes `/metrics` | False as stated. `web /metrics` is `404`, and simulator curl failed on `8086` | `README.md:38-42`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_metrics.txt`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/simulator_metrics.txt` |
| RBAC on mutation endpoints | README says mutation RBAC exists | Mostly true for tested write paths: unauth `401`, viewer `403` for alert ack; governance write routes are wrapped in authz middleware | `README.md:32-36`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/alerts_ack_unauth.txt`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/alerts_ack_viewer.txt`; `cmd/alerts/main.go:116-133`; `cmd/governance/main.go:78-80`, `cmd/governance/main.go:127-153` |
| CSRF enforced | README and docs imply CSRF is in place | Partial. Middleware exists, but unauthenticated mutating requests bypass it and authenticated logout succeeded without `X-CSRF-Token` | `internal/middleware/csrf.go:61-84`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/gateway_logout_no_csrf.txt`; `cmd/gateway-api/main.go:71-77`, `cmd/gateway-api/main.go:115-123` |
| UI live updates via web proxy | README says SSE live updates via web proxy routes | Partial and degraded. SSE shell exists, but current proxy auth failure yields degraded events with `upstream 401` | `README.md:44-49`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/sse_queue.txt` |
| Operator console login demo works | README demo steps imply UI login flow works | False in current browser proof. Playwright stayed on `/login` instead of redirecting to `/command-center` | `README.md:136-143`; `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt` |
| Quant lineage + calibration are real | `docs/PRODUCTION_READINESS_REVIEW.md` claims this was fixed | True. Inference publishes lineage, trust exposes calibration, and `/readyz` shows active bundle metadata | `docs/PRODUCTION_READINESS_REVIEW.md:3-8`; `services/inference/app.py:204-236`; `services/inference/app.py:256-270`; `internal/query/trust.go:24-79`; `docs/progress_check/20260301T165141Z/EVIDENCE/metrics/inference_readyz.json` |

## System Map

### Services

- `gateway-api`
  - Entrypoint: `cmd/gateway-api/main.go:31-144`
  - Routes: `/`, `/healthz`, `/readyz`, `/metrics`, `/auth/login`, `/auth/logout`, `/me` (`cmd/gateway-api/main.go:52-133`)
  - Role: JWT cookie auth, login throttling, CSRF bootstrap
- `aggregator`
  - Entrypoint: `cmd/aggregator/main.go`
  - Role: consumes `raw.ticks`, writes `raw_ticks` and `candles`, produces `derived.candles`
- `features`
  - Entrypoint: `cmd/features/main.go`
  - Role: computes feature vectors and DQ, writes `features`, produces `derived.features` and `dq.metrics`
  - Key non-atomic path: writes DB, then publishes Kafka without a shared transaction (`cmd/features/main.go:175-181`)
- `inference`
  - Entrypoint: `services/inference/app.py`
  - Role: consumes feature messages, scores anomaly/escalation, emits lineage-rich score payloads (`services/inference/app.py:183-243`)
- `alerts`
  - Entrypoint: `cmd/alerts/main.go`
  - Role: consumes `derived.scores`, upserts `scores`, creates `alerts` and `incidents`, exposes write APIs and SSE
  - Score persistence: `cmd/alerts/main.go:467-482`
- `governance`
  - Entrypoint: `cmd/governance/main.go`
  - Role: model deploys, replay starts, policy administration, forecast refresh
- `query`
  - Entrypoint: `cmd/query/main.go`
  - Role: read models for queue, command center, feed, trust, world map, scores, candles
  - Read-gated debug route: `/debug/seed-status` now requires `read` (`cmd/query/main.go:128-138`)
- `web`
  - Next.js frontend
  - Proxy route: `web/app/api/proxy/[service]/[...path]/route.ts:3-48`
  - App shell and filters: `web/components/AppShell.tsx:12-181`

### Topics

- `raw.ticks`
- `derived.candles`
- `derived.features`
- `derived.scores`
- `alerts.created`
- `dq.metrics`

Proof: `docs/progress_check/20260301T165141Z/EVIDENCE/docker/topics.txt`

### Tables

Primary live tables with non-zero data:

- `raw_ticks`
- `candles`
- `features`
- `scores`
- `alerts`
- `incidents`

Proof: `docs/progress_check/20260301T165141Z/EVIDENCE/sql/counts.txt`

### Frontend endpoints and UI wiring

- Proxy: `web/app/api/proxy/[service]/[...path]/route.ts:26-39`
- SSE routes: `docs/progress_check/20260301T165141Z/INVENTORY/web_routes.txt`
- Filter state:
  - parse/serialize: `web/lib/filterState.ts:44-139`
  - shell application + URL replacement: `web/components/AppShell.tsx:27-43`
  - filter strip rendering: `web/components/AppShell.tsx:155-181`
- Feed modes:
  - `For You` / `Trending`: `web/app/feed/page.tsx` (see `docs/progress_check/20260301T165141Z/INVENTORY/ui_refs.txt`)

## Math Validation

### Located scoring formulas

- Z-score clipping: `internal/marketmath/zscore.go:3-15`
  - Bounded to `[-8, 8]`
- EWMA update and volatility:
  - `internal/marketmath/ewma.go:15-25`
  - `internal/marketmath/volatility.go:5-13`
- Feature generation:
  - `cmd/features/main.go:196-240`
- Inference scoring:
  - anomaly: `services/inference/src/scoring.py:8-21`
  - escalation: `services/inference/src/scoring.py:24-52`
  - composite risk: `services/inference/src/scoring.py:74-80`
  - priority score: `services/inference/src/scoring.py:89-94`

### Sanity assessment

- Z-score is explicitly clipped, so it cannot explode numerically (`internal/marketmath/zscore.go:10-14`).
- `composite_risk` is designed as a bounded weighted mix and is emitted both as `score` and `composite` in the score payload (`services/inference/app.py:223-240`).
- Severity banding is tied to escalation output, but the exact runtime monotonicity for every feature combination was not executed in this audit. `UNKNOWN`: no dedicated backtest or property-test output was run in this pass.
- Calibration is present in bundle metadata and exposed via trust, but no separate recalibration job was executed in this audit. `UNKNOWN`: runtime backtest command output is missing.

## Production Readiness Rubric

Demo completion uses the repo’s own audit markers. Production readiness uses the stricter weighted rubric below.

### Demo Completion %

- Score: `100%`
- Basis: latest gate artifact reports all ten stage markers present (`docs/audit/20260301T163152Z/RESULT.md`)
- Important caveat: the same run’s traceability file still includes stale skip language, so this is not a trustworthy production proxy (`docs/audit/20260301T163152Z/TRACEABILITY.md`)

### Production Readiness %

Weighted total: **68.4%**

| Category | Weight | Score | Weighted Points | Evidence |
| --- | ---: | ---: | ---: | --- |
| Backend APIs | 12 | 74 | 8.88 | `cmd/gateway-api/main.go:71-133`, `cmd/alerts/main.go:116-172`, `cmd/governance/main.go:78-153`, security curl proofs |
| Data pipeline | 12 | 76 | 9.12 | `docs/progress_check/20260301T165141Z/EVIDENCE/sql/counts.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/docker/topics.txt`, `cmd/features/main.go:175-181`, `cmd/alerts/main.go:467-488` |
| Database maturity | 10 | 78 | 7.80 | migrations in `sql/migrations`, `docs/progress_check/20260301T165141Z/EVIDENCE/sql/tables.txt`, live non-zero tables |
| ML/Quant stack | 14 | 84 | 11.76 | `docs/progress_check/20260301T165141Z/EVIDENCE/metrics/inference_readyz.json`, `services/inference/app.py:183-270`, `internal/query/trust.go:24-79` |
| UI/UX | 12 | 56 | 6.72 | `web/components/AppShell.tsx:27-181`, `web/lib/filterState.ts:44-139`, `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_proxy_login_raw.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt` |
| Observability | 12 | 72 | 8.64 | `internal/metrics/metrics.go:17-138`, `docs/progress_check/20260301T165141Z/EVIDENCE/metrics/go_metrics_summary.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/metrics/prom_up.json` |
| Security | 10 | 62 | 6.20 | `internal/authz/middleware.go:56-85`, `internal/middleware/csrf.go:61-84`, security curl proofs |
| Testing | 10 | 38 | 3.80 | `docs/progress_check/20260301T165141Z/EVIDENCE/tests/cover_func.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/tests/web_npm_test.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt` |
| Production ops | 8 | 68 | 5.44 | `docs/progress_check/20260301T165141Z/EVIDENCE/docker/compose_ps.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/docker/migrate_tail.txt`, web-container DNS proof |

### End-user readiness (informational, not part of weighted total)

- Score: `54/100`
- Rationale: the backend and data are live, but the primary user path is currently compromised by web-proxy auth failure and Playwright login failure (`docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_proxy_login_raw.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt`)

## Milestones Reached

- Deterministic-enough Compose boot succeeded in this run (`docs/progress_check/20260301T165141Z/EVIDENCE/docker/compose_up.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/docker/compose_ps.txt`)
- Score persistence is live (`docs/progress_check/20260301T165141Z/EVIDENCE/sql/counts.txt`)
- Quant lineage is live (`services/inference/app.py:204-236`, `cmd/alerts/main.go:467-470`)
- Trust/calibration surface is live (`internal/query/trust.go:56-79`)
- Prometheus and Grafana are up and provisioned (`docs/progress_check/20260301T165141Z/EVIDENCE/metrics/prom_up.json`, `docs/progress_check/20260301T165141Z/EVIDENCE/metrics/grafana_health.json`)
- Web unit tests are green (`docs/progress_check/20260301T165141Z/EVIDENCE/tests/web_npm_test.txt`)

## Blockers

### Critical

1. Web proxy login is failing, which breaks the main end-user path.
   - Proof: `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_proxy_login_raw.txt`
   - Impact: authenticated UI fetches and SSE degrade to upstream `401`
2. CSRF is not being enforced on at least one authenticated mutating route (`/auth/logout`).
   - Proof: `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/gateway_logout_no_csrf.txt`
   - Code path: `internal/middleware/csrf.go:73-82`, `cmd/gateway-api/main.go:115-123`
3. E2E browser proof is failing at login.
   - Proof: `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt`

### Major

1. Go integration coverage is too low for production confidence.
   - Proof: `docs/progress_check/20260301T165141Z/EVIDENCE/tests/cover_func.txt`
2. README overstates metrics coverage.
   - Proof: `README.md:38-42`, `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_metrics.txt`, `docs/progress_check/20260301T165141Z/EVIDENCE/curl/simulator_metrics.txt`
3. Rate limiting was not observed in runtime probe.
   - Proof: `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/login_rate_limit_codes.txt`

### Minor

1. `docs/PRODUCTION_READINESS_REVIEW.md` still contains stale claims that conflict with current runtime and source.
   - Proof: `docs/PRODUCTION_READINESS_REVIEW.md:38-66`, `docs/PRODUCTION_READINESS_REVIEW.md:82-85`, `docs/PRODUCTION_READINESS_REVIEW.md:123-128`
2. `docs/Asset-Manager` contains an embedded `.git` directory and looks like vendor/reference ballast.
   - Proof: `docs/progress_check/20260301T165141Z/INVENTORY/asset_manager_files.txt`

## What To Do Next

### 1 day

- Fix and re-prove the web proxy login path, then rerun Playwright login.
- Re-run CSRF checks with explicit authenticated mutations beyond logout.
- Re-run rate-limit proof until either `429` appears or the missing middleware application point is identified.

### 3 days

- Add integration tests for the zero-covered orchestration paths:
  - `alerts.consumeScores`
  - `query.main` route wiring
  - `governance.refreshForecast`
  - `internal/incidents.*`
- Persist Playwright traces/screenshots to the host workspace so browser failures are inspectable in review artifacts.

### 7 days

- Raise Go coverage on critical paths above a defensible floor.
- Align README/doc claims with runtime truth, especially metrics and UI-login expectations.
- Audit the entire web proxy auth flow from Next route to gateway cookie handling and close the current end-user gap.
