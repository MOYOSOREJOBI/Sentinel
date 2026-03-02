# Sentinel Forensic Audit Scorecard

- Audit scope: current working tree at `cd4310ec3dc498f6272e798696b6604e429a5586`, not a clean commit. See `docs/review/20260301T041338Z/git_head.txt` and `docs/review/20260301T041338Z/git_status.txt`.
- Working tree is dirty, so findings apply to the checked-out source plus uncommitted changes now present. See `docs/review/20260301T041338Z/git_status.txt`.
- External blueprint inputs under `/mnt/data` were unavailable in this environment and are therefore `UNPROVEN` inputs.
- `docs/PRODUCTION_READINESS_REVIEW.md` is internally contradictory and is treated as a claim ledger, not ground truth. It claims Quant/ML upgrades at `docs/PRODUCTION_READINESS_REVIEW.md:3-8`, but stale sections still claim no score persistence at `docs/PRODUCTION_READINESS_REVIEW.md:18` and `docs/PRODUCTION_READINESS_REVIEW.md:31-36`.
- Existing audit gates are not treated as proof. `docs/audit/_latest/RESULT.md:1-4` says `PASS` with `RUNTIME COVERAGE: FULL`, while `docs/audit/_latest/TRACEABILITY.md:17-19` still documents Docker and Playwright skip markers.

## 1) Backend APIs (0–100)

- Score: **52/100**
- Auth correctness: `UNPROVEN` at runtime after clean boot because `gateway-api`, `query`, `alerts`, and `governance` never started. `docs/review/20260301T041338Z/compose_ps.txt` and `docs/review/20260301T041338Z/runtime_routes/gateway_login.txt`.
- Source truth is materially better than the stale review doc. `alerts` mutations require auth plus RBAC in `cmd/alerts/main.go:121-135`, `cmd/alerts/main.go:137-375`. `governance` mutations require auth plus RBAC in `cmd/governance/main.go:135-206`.
- Query read routes are guarded by `requireRole(pub, "read")` in `cmd/query/main.go:137-138`, then all major read endpoints (`/queue`, `/command-center`, `/feed`, `/search-suggest`, `/trust`, `/world-map`, `/scores`, `/candles`) are mounted inside that protected group in `cmd/query/main.go:139-520`.
- There is still at least one unauthenticated diagnostic route: `/debug/seed-status` is mounted before the auth-protected group in `cmd/query/main.go:129-135`.
- There is still at least one unauthenticated live read surface: `/sse/alerts` is mounted without auth in `cmd/alerts/main.go:118-120`.
- “Unauth governance endpoints” status: **UNPROVEN at runtime, source indicates PASS for write paths**. The clean-boot runtime probe could not reach governance because the service never came up; see `docs/review/20260301T041338Z/runtime_routes/governance_models_deploy_unauth.txt`. In source, `/models/deploy` is guarded in `cmd/governance/main.go:135-163`.
- “401 truth, no masking” status: **FAIL in source ergonomics / UNPROVEN in runtime**. Auth failures are often returned as `403 forbidden` on combined auth+RBAC checks, e.g. `cmd/alerts/main.go:122-125`, not a clean unauthenticated `401` split.

## 2) Data pipeline (0–100)

- Score: **24/100**
- Clean-boot determinism: **FAIL**. The stack fails at migration 008 during a fresh `down -v` then `up -d --build`. `docs/review/20260301T041338Z/migrate.log` shows `ERROR: cannot create a unique index without the column "ts"`, triggered by `ALTER TABLE scores ADD CONSTRAINT scores_id_pk PRIMARY KEY (id)` from `sql/migrations/008_scores_lineage.sql:2-12`.
- The failure is structural, not transient. `scores` is a hypertable partitioned on time, so adding a primary key on `id` alone is invalid in Timescale/Postgres. The exact failing statement is in `sql/migrations/008_scores_lineage.sql:10-11`.
- Boot dependency ordering is otherwise improved. `migrate` waits for Postgres health at `deploy/docker/docker-compose.yml:36-67`, and `topics-init` runs before producers at `deploy/docker/docker-compose.yml:68-83`.
- Kafka topic race mitigation is partly real. `simulator` depends on `topics-init` completion at `deploy/docker/docker-compose.yml:101-106`, and all expected topics exist after boot; see `docs/review/20260301T041338Z/topics_list.txt`.
- Most runtime services are transitively blocked on `migrate: service_completed_successfully`, so `gateway-api`, `aggregator`, `features`, `alerts`, `governance`, `query`, and `web` never start when 008 fails. See `deploy/docker/docker-compose.yml:84-99`, `114-146`, `160-203`, `205-233`, and `docs/review/20260301T041338Z/compose_ps.txt`.
- Score persistence truth on this clean boot: **FAIL**. All core pipeline tables are zero, including `scores=0`; see `docs/review/20260301T041338Z/db_counts.txt`.
- Exactly-once-ish semantics are only partial in source. `alerts` upserts `scores` by `(idempotency_key, ts)` in `cmd/alerts/main.go:452-464`, but this path never executed in the clean boot because `alerts` was not running.

## 3) Quant/ML (0–100)

- Score: **58/100**
- Real trained models deployed: **PASS in source and running inference container**. The model bundle exists in `services/inference/models/` and is confirmed by `docs/review/20260301T041338Z/model_bundle_ls.txt`.
- Fallback status: **PASS in the inference container**. `/readyz` inside the container returns `fallback_mode:false`, `model_version:"quant-v1-2026-03-01"`, and a non-empty artifact hash; see `docs/review/20260301T041338Z/runtime_routes/inference_readyz_container.txt`. The response contract is implemented in `services/inference/app.py:256-271`.
- Lineage emission in produced score messages is real in source. Inference emits `feature_snapshot_hash`, `model_version`, `artifact_hash`, `model_artifact_hash`, `scoring_run_id`, and `produced_at` in `services/inference/app.py:203-243`.
- Score persistence code path is real in source. `alerts` consumes `derived.scores`, extracts lineage fields, and upserts them into `scores` in `cmd/alerts/main.go:393-464`.
- `/scores` is wired to return lineage when data exists. `internal/query/scores.go:12-29` defines lineage-bearing response fields, and `internal/query/scores.go:70-99` selects `produced_at`, `scoring_run_id`, `artifact_hash`, `model_artifact_hash`, and `feature_snapshot_hash`.
- Quant runtime end-to-end is still **FAIL on clean boot**. Because migration 008 blocks `aggregator`, `features`, `alerts`, and `query`, the pipeline never warms and `scores=0`; see `docs/review/20260301T041338Z/db_counts.txt` and `docs/review/20260301T041338Z/compose_ps.txt`.
- Calibration metrics do exist in the inference service. Gauges for `inference_escalation_auc`, `inference_escalation_brier`, `inference_escalation_pr_auc`, and `inference_anomaly_stability_score` are defined in `services/inference/app.py:48-51` and exposed with non-zero values in `docs/review/20260301T041338Z/runtime_routes/inference_metrics_container.txt`.
- Backtest/live processing truth is still weak in this boot. Inference is up, but its live counters are zero: `inference_messages_consumed_total 0`, `inference_messages_produced_total 0`, `inference_scoring_latency_seconds_count 0`; see `docs/review/20260301T041338Z/runtime_routes/inference_metrics_container.txt` and `docs/review/20260301T041338Z/prom_queries/inference_messages_consumed_total_internal.json`.

## 4) UI/UX (0–100)

- Score: **33/100**
- Runtime UI proof is largely `UNPROVEN` after clean boot because `web` and `query` never started. The host check for `http://localhost:3000/login` failed with connection refused; see `docs/review/20260301T041338Z/playwright/web_login_host_check.txt`.
- Filters are not URL-canonical. Global filters are stored in `localStorage`, not parsed from and serialized to the URL, in `web/components/AppShell.tsx:23-41`.
- Filter controls exist and feed component state, but end-to-end refetch under a clean boot could not be proven because the UI was down. The control surface is in `web/components/AppShell.tsx:169-198`.
- Typeahead exists but is narrower than the requested product bar. It is a debounced `datalist` backed by `/search-suggest` in `web/components/AppShell.tsx:76-129` and `cmd/query/main.go:200-285`, not a richer combobox with explicit per-field query shaping.
- Custom date range still uses `datetime-local` strings in `web/components/AppShell.tsx:182-185`; backend parsing and runtime behavior were not revalidated in this audit because `query` never came up.
- Charts are real but incomplete against the requested “multi-scale” bar. `lightweight-charts` is used in `web/components/CandleRiskPanel.tsx:24-59`, but the component fetches only `api.candles(symbol, window, 300)` and `api.scores(symbol, window, 300)` in `web/components/CandleRiskPanel.tsx:83-86`, with no resolution selector.
- Backend chart API also lacks a resolution parameter. `/candles` accepts `symbol`, `window`, and `maxPoints`, but no `res`, in `cmd/query/main.go:486-503`.
- The globe is interactive, not decorative: Three.js raycasting and click-to-filter are implemented in `web/components/RiskGlobe.tsx:111-165`. But country detection falls back to coarse bounding boxes in `web/components/RiskGlobe.tsx:7-45`, not polygon/UV lookup.
- Feed UX is more than placeholder code. It includes `Default`, `Trending`, and `For You`, infinite scroll, SSE-triggered refresh, and polling fallback in `web/app/feed/page.tsx:7-84`. Runtime validation of these flows is still `UNPROVEN` in the clean boot because the UI never came up.

## 5) Observability (0–100)

- Score: **39/100**
- Go service metrics are still mostly stubbed. `internal/metrics/metrics.go:8-12` exposes only `sentinel_up 1`.
- The expected real Go HTTP metric surface is absent. Querying `http_requests_total` returns an empty vector in `docs/review/20260301T041338Z/prom_queries/http_requests_total_internal.json`.
- Prometheus is running internally and scraping, but most Go jobs are down because the services never started after migration failure. `docs/review/20260301T041338Z/prom_queries/up_internal.json` shows `inference=1`, while `gateway`, `features`, `alerts`, `governance`, `query`, and `aggregator` are all `0`.
- Inference observability is real. The service exports counters, histograms, fallback mode, and calibration gauges in `services/inference/app.py:37-51`, and these appear in `docs/review/20260301T041338Z/runtime_routes/inference_metrics_container.txt`.
- Grafana provisioning is real. Internal API health returns healthy in `docs/review/20260301T041338Z/prom_queries/grafana_health_internal.json`, and the `Sentinel Ops` dashboard is discoverable in `docs/review/20260301T041338Z/prom_queries/grafana_search_internal.json`.
- Host-level Prometheus and Grafana reachability from this shell is `UNPROVEN`/degraded. Direct curls to `localhost:9090` and `localhost:3001` failed even though the containers are up; see `docs/review/20260301T041338Z/prom_up.json`, `docs/review/20260301T041338Z/grafana_health.json`, and `docs/review/20260301T041338Z/grafana_search.json`.

## 6) Production hardening (0–100)

- Score: **20/100**
- Deterministic migrations: **FAIL**. A clean boot fails consistently at migration 008, preventing the application from reaching a runnable state; see `docs/review/20260301T041338Z/migrate.log`.
- Recoverability is weak because the entire application plane is chained behind `migrate: service_completed_successfully`; once 008 fails, nearly all user-facing services remain down. See `deploy/docker/docker-compose.yml:84-99`, `114-146`, `160-203`, `205-233`, and `docs/review/20260301T041338Z/compose_ps.txt`.
- Security baseline in source is partial but not sufficient to offset boot failure. Sensitive writes are mostly RBAC-wrapped in `cmd/alerts/main.go:121-135` and `cmd/governance/main.go:135-206`, but `/sse/alerts` and `/debug/seed-status` remain open in `cmd/alerts/main.go:118-120` and `cmd/query/main.go:129-135`.
- Secrets/TLS posture is only locally acceptable. Compose runs dev credentials and local key paths directly in environment variables at `deploy/docker/docker-compose.yml:86`, `162`, `179`.
- The requested production proof scenarios are not met after clean boot. There is no stable end-user path because `web`, `query`, `gateway-api`, `alerts`, and `governance` are absent from the live process list in `docs/review/20260301T041338Z/compose_ps.txt`.

## Overall %

- Weighted formula: UI/UX 25, Quant/ML 25, Data 20, Backend 15, Observability 10, Production hardening 5.
- Weighted overall score: **40/100**
- Calculation: `(33*0.25) + (58*0.25) + (24*0.20) + (52*0.15) + (39*0.10) + (20*0.05) = 40.25`
- Production reality verdict: **Not production-ready**. The clean boot fails before the user-facing stack can start, and the current runtime cannot prove core end-to-end product behavior.

## Top 10 blockers (smallest fix first)

1. **Migration 008 adds an invalid primary key on a hypertable**
- Severity: Critical
- Why it reduces the score: It breaks clean boot and blocks almost all runtime proof.
- Proof: `docs/review/20260301T041338Z/migrate.log`
- Source: `sql/migrations/008_scores_lineage.sql:2-12`
- Proof target: clean `up -d --build` completes and `migrate` exits `0`.

2. **Fresh-boot pipeline never warms (`scores=0`, `features=0`, `alerts=0`)**
- Severity: Critical
- Why it reduces the score: Data, Quant/ML, and UI claims are unprovable in the required clean-boot path.
- Proof: `docs/review/20260301T041338Z/db_counts.txt`
- Source: runtime blocked by `deploy/docker/docker-compose.yml:84-99`, `114-146`, `160-203`
- Proof target: non-zero counts after a clean boot and warm period.

3. **User-facing services are chained behind migration success**
- Severity: Critical
- Why it reduces the score: A single migration defect removes the entire product surface.
- Proof: `docs/review/20260301T041338Z/compose_ps.txt`
- Source: `deploy/docker/docker-compose.yml:84-99`, `114-146`, `160-203`, `205-233`
- Proof target: `gateway-api`, `query`, `alerts`, and `web` all healthy after clean boot.

4. **`/sse/alerts` is exposed without auth**
- Severity: Major
- Why it reduces the score: Sensitive live alert data is readable without authentication.
- Proof: source-only in this audit because runtime service was down
- Source: `cmd/alerts/main.go:118-120`
- Proof target: unauthenticated request gets `401` or `403`.

5. **`/debug/seed-status` is exposed without auth**
- Severity: Major
- Why it reduces the score: Internal readiness diagnostics are publicly reachable in source.
- Proof: source-only in this audit because runtime service was down
- Source: `cmd/query/main.go:129-135`
- Proof target: route is removed or protected by auth.

6. **Authn and authz are conflated into `403` responses**
- Severity: Major
- Why it reduces the score: It obscures unauthenticated vs unauthorized failures and weakens auth correctness claims.
- Proof: source-only in this audit because runtime service was down
- Source: `cmd/alerts/main.go:122-125`
- Proof target: anonymous callers get `401`; wrong role gets `403`.

7. **Go observability is still `sentinel_up`-only**
- Severity: Major
- Why it reduces the score: Dashboarded read/write health cannot be trusted without real request and latency metrics.
- Proof: `docs/review/20260301T041338Z/prom_queries/http_requests_total_internal.json`
- Source: `internal/metrics/metrics.go:8-12`
- Proof target: non-empty `http_requests_total` and request latency histograms.

8. **Host-level Prometheus and Grafana reachability is inconsistent**
- Severity: Major
- Why it reduces the score: Local operability from the host shell is degraded even while containers report up.
- Proof: `docs/review/20260301T041338Z/prom_up.json`, `docs/review/20260301T041338Z/grafana_health.json`
- Source: runtime artifact mismatch, not a single source file
- Proof target: host curls to `localhost:9090` and `localhost:3001` return valid JSON.

9. **UI filter state is not URL-canonical**
- Severity: Major
- Why it reduces the score: Refresh/shareable state and deterministic replay of user context are weak.
- Proof: source-only because `web` was down
- Source: `web/components/AppShell.tsx:23-41`
- Proof target: filter changes update URL and survive refresh.

10. **Charts do not support resolution switching**
- Severity: Minor
- Why it reduces the score: The UI misses a key “real terminal” behavior and the backend API has no `res` support.
- Proof: source-only because `web` and `query` were down
- Source: `web/components/CandleRiskPanel.tsx:83-86`, `cmd/query/main.go:486-503`
- Proof target: `/candles?res=...` works and the UI exposes resolution controls.

## What to build next (7 days)

1. **Day 1:** Re-run the same clean-boot audit after resolving the migration-008 hypertable key issue. The goal is to move Data and Hardening out of immediate boot-failure territory with a new `compose_ps.txt`, `migrate.log`, and `db_counts.txt`.
2. **Day 2:** Re-prove score persistence on a fresh boot. Capture non-zero `scores`, then save `/scores` output showing lineage fields from the live `query` service.
3. **Day 3:** Re-run auth probes against live `gateway-api`, `alerts`, and `governance`, specifically separating anonymous vs wrong-role behavior. This moves Backend from source-backed to runtime-backed.
4. **Day 4:** Re-run Playwright once `web` and `query` are healthy. Prove filter changes alter request params and returned payloads, then re-score UI/UX on runtime evidence instead of source-only evidence.
5. **Day 5:** Add a second observability pass that proves real non-stub Go metrics or confirms they are still missing. Capture Prometheus query outputs for request counters and latency histograms.
6. **Day 6:** Reconcile dashboard truth with Prometheus truth. Save Grafana search plus targeted PromQL evidence showing which panels have non-empty data and which remain cosmetic.
7. **Day 7:** Re-audit the whole stack from `down -v` again and compare section scores only where new runtime artifacts materially changed confidence. The goal is to turn current `UNPROVEN` items into proven pass/fail outcomes.
