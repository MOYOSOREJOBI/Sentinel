1) Backend APIs (0–100)
- External blueprint inputs under `/mnt/data` are unavailable here, so any blueprint-based expectations are `UNPROVEN`; see `docs/review/20260301T054050Z/mnt_data_unavailable.txt`.
- This audit applies to the dirty working tree at `cd4310ec3dc498f6272e798696b6604e429a5586`, not a clean commit; see `docs/review/20260301T054050Z/git_head.txt` and `docs/review/20260301T054050Z/git_status.txt`.
- Auth correctness (401 truth, no masking): mixed. Login currently returns `401`, but the clean boot seeds zero users, so authenticated happy-path validation is `UNPROVEN`; see `docs/review/20260301T054050Z/runtime_routes/internal_auth_http.txt` and `docs/review/20260301T054050Z/db_users.txt`.
- RBAC integrated (not just “package exists”): yes in current source. Query wraps its main read surface with `requireRole(pub, "read")` at `cmd/query/main.go:137-138`, then mounts `/queue`, `/command-center`, `/feed`, `/search-suggest`, `/trust`, `/world-map`, `/scores`, and `/candles` inside that group at `cmd/query/main.go:139-504`. Alerts write routes gate on `rbac.Allowed(...)` at `cmd/alerts/main.go:121-135`, `cmd/alerts/main.go:137-375`. Governance gates read and write routes at `cmd/governance/main.go:83-317` and is corroborated by `docs/review/20260301T054050Z/rbac_refs.txt`.
- “unauth governance endpoints” status: `PASS` for the write path tested. Unauthenticated `POST /models/deploy` returns `403 forbidden` in the live stack; see `docs/review/20260301T054050Z/runtime_routes/internal_auth_http.txt`. Source check aligns: `cmd/governance/main.go:137-163`.
- Two unauthenticated surfaces remain. `/debug/seed-status` is mounted outside auth in `cmd/query/main.go:129-135` and returns `200` with live counts in `docs/review/20260301T054050Z/runtime_routes/internal_basic_http.txt`. `/sse/alerts` is also mounted without auth at `cmd/alerts/main.go:118-120`.
- Auth error semantics are still coarse. Unauthenticated alert ack returns `403 forbidden`, not `401`, in `docs/review/20260301T054050Z/runtime_routes/internal_auth_http.txt`, matching the combined `!ok || !Allowed(...)` branch at `cmd/alerts/main.go:122-125`.
- Score: **68/100**

2) Data pipeline (0–100)
- Boot determinism (topics/init races): materially improved and currently passes on a clean boot. `migrate` waits for Postgres readiness at `deploy/docker/docker-compose.yml:36-67`; `topics-init` waits for Redpanda and creates the required topics at `deploy/docker/docker-compose.yml:68-83`; `simulator` depends on `topics-init` completion at `deploy/docker/docker-compose.yml:101-106`.
- Clean boot result: `PASS` in this run. All core services are healthy and `migrate` exited cleanly; see `docs/review/20260301T054050Z/compose_ps.txt` and `docs/review/20260301T054050Z/migrate.log`.
- Exactly-once-ish semantics (idempotency keys + txn order): partial. `alerts` persists scores with `INSERT ... ON CONFLICT (idempotency_key, ts) DO UPDATE` at `cmd/alerts/main.go:452-464`, but the score write, alert creation, and Kafka publish are not wrapped in one SQL transaction; see `cmd/alerts/main.go:463-479`.
- Score persistence truth: `PASS`. Current DB counts show non-zero `raw_ticks`, `candles`, `features`, and `scores`; see `docs/review/20260301T054050Z/db_counts_actual.txt`. The runtime writer exists in `cmd/alerts/main.go:393-464`, and replay also writes scores in `internal/replay/runner.go:70` as captured in `docs/review/20260301T054050Z/scores_refs.txt`.
- Current pipeline asymmetry: `alerts=0` and `incidents=0` while `scores` is large, so the scoring sink is working even when alert promotion is not firing in this warm period; see `docs/review/20260301T054050Z/db_counts_actual.txt`.
- Kafka topic state is correct in the running stack: `raw.ticks`, `derived.candles`, `derived.features`, `derived.scores`, and `alerts.created` all exist; see `docs/review/20260301T054050Z/topics_list.txt`.
- Score: **74/100**

3) Quant/ML (0–100)
- Real trained models deployed? `PASS`. The repo contains a concrete model bundle under `services/inference/models/`, and bundled metadata exposes `model_version`, `trained_at`, `artifact_hash`, and calibration metrics; see `services/inference/models/metadata.json` (captured earlier in local source) and `docs/review/20260301T054050Z/PRODUCTION_READINESS_REVIEW_excerpt.txt:3-8`.
- Inference is not in permanent fallback: `PASS`. Live `/readyz` from the running inference container reports `fallback_mode:false`, `model_version:"quant-v1-2026-03-01"`, and a non-empty `model_artifact_hash`; see `docs/review/20260301T054050Z/runtime_routes/internal_basic_http.txt`. Source contract is at `services/inference/app.py:256-271`.
- Lineage: `PASS` in both message construction and persisted rows. Inference emits `feature_snapshot_hash`, `artifact_hash`, `model_artifact_hash`, `scoring_run_id`, `produced_at`, and `model_version` at `services/inference/app.py:203-243`. Alerts persists those fields into `scores` at `cmd/alerts/main.go:411-417` and `cmd/alerts/main.go:463-464`.
- Persisted lineage truth: `PASS`. `docs/review/20260301T054050Z/db_lineage.txt` shows `49,979` rows with non-null lineage and sample rows that include `produced_at`, `model_version`, `scoring_run_id`, `feature_snapshot_hash`, `artifact_hash`, and `model_artifact_hash`.
- Calibration/backtests exist with stored metrics: `PASS` for stored model-bundle metrics, `partial` for runtime backtest proof. Inference publishes `inference_escalation_auc`, `inference_escalation_brier`, `inference_escalation_pr_auc`, and `inference_anomaly_stability_score` at `services/inference/app.py:48-51` and syncs them at `services/inference/app.py:124-144`. Trust surfaces the same metrics from model bundle metadata at `internal/query/trust.go:65-79`.
- `/scores` API data under authenticated user is `UNPROVEN` in this clean boot because there are zero users and login returns `401`; see `docs/review/20260301T054050Z/db_users.txt` and `docs/review/20260301T054050Z/runtime_routes/internal_auth_http.txt`. The query path itself is wired to `scores` and returns lineage-rich points in source at `cmd/query/main.go:467-485` and `internal/query/scores.go:62-117`.
- Score: **80/100**

4) UI/UX (0–100)
- Filters: partial. The UI calls backend endpoints with filter query params through `filtersToQuery(...)` in `web/lib/api.ts:30-47`, but global filter state itself is stored in `localStorage`, not URL state, in `web/components/AppShell.tsx:23-41`. This means refresh/shareable-state correctness is incomplete.
- Typeahead: partial but real. A 150ms debounced suggestion fetch exists in `web/components/AppShell.tsx:76-92`, and `/search-suggest` resolves `symbol`, `country`, `region`, `sector`, `industry`, and `venue` in `cmd/query/main.go:200-285`. The UI still uses a plain `datalist`, not a richer navigable combobox, at `web/components/AppShell.tsx:188-191`.
- Charts: real but not full “terminal-grade.” `lightweight-charts` is used for candles plus one risk overlay in `web/components/CandleRiskPanel.tsx:24-59`, and data is fetched from live `/candles` and `/scores` in `web/components/CandleRiskPanel.tsx:83-86`. There is no resolution selector, and `/candles` supports no `res` parameter in `cmd/query/main.go:486-503`.
- Globe: interactive but simplified. Three.js raycasting, hover, and click-to-filter are real in `web/components/RiskGlobe.tsx:111-165`, and Command Center applies the selected country filter at `web/app/command-center/page.tsx:97-99`. Country lookup still uses hard-coded bounding boxes, not real polygon/UV mapping, at `web/components/RiskGlobe.tsx:7-45`.
- Feed: partial but materially implemented. `/feed` supports `Default`, `Trending`, and `For You` modes in `web/app/feed/page.tsx:7-13`, uses infinite scroll in `web/app/feed/page.tsx:74-84`, and retries via SSE/polling in `web/app/feed/page.tsx:55-72`. The live UI path is still `UNPROVEN` in this audit because authenticated browser navigation is blocked by the empty `users` table.
- Real-time: present in source. Command Center uses `useSSE` with dedupe and rate-limiting at `web/app/command-center/page.tsx:52-68`; the generic SSE hook reconnects with backoff in `web/lib/useSSE.ts:11-41`. Some pages still show only binary “live / polling” state, not a full connected/degraded/offline tri-state, e.g. `web/app/feed/page.tsx:133-135`.
- Runtime browser proof: `UNPROVEN`. No Playwright artifact was captured in this pass, and authenticated end-user navigation could not be established because `users_count=0`; see `docs/review/20260301T054050Z/db_users.txt`.
- Score: **45/100**

5) Observability (0–100)
- Do Go services emit real metrics or only `sentinel_up`? They still emit only `sentinel_up`. The handler is a static text payload in `internal/metrics/metrics.go:8-12`, and live query `/metrics` returns only that stub in `docs/review/20260301T054050Z/runtime_routes/internal_basic_http.txt`.
- Do Grafana panels show real data? Mixed. The dashboard exists and is provisioned (`Sentinel Ops`), proven by `docs/review/20260301T054050Z/runtime_routes/internal_obs_http.txt` and mirrored in `grafana/dashboards/sentinel-dashboard.json:1-86`.
- Prometheus scrape health is real in the running stack: `up` is `1` for gateway, aggregator, features, alerts, inference, governance, and query; see `docs/review/20260301T054050Z/runtime_routes/internal_obs_http.txt`.
- Inference observability is real and non-stubbed. Inference defines counters, histograms, fallback gauges, and calibration gauges at `services/inference/app.py:37-51`, and the live Prometheus query returns `inference_fallback_mode=0`; see `docs/review/20260301T054050Z/runtime_routes/internal_obs_http.txt`.
- The cross-service HTTP metric contract is still missing. Prometheus returns an empty vector for `http_requests_total`; see `docs/review/20260301T054050Z/runtime_routes/internal_obs_http.txt`, which matches `docs/review/20260301T054050Z/metrics_refs.txt` showing no real `client_golang` request instrumentation in Go services.
- Host-port observability from this shell remains broken or sandbox-blocked. Direct host curls to `localhost:9090` and `localhost:3001` fail, while the same services work from inside the `web` container; see `docs/review/20260301T054050Z/prom_up.json`, `docs/review/20260301T054050Z/grafana_health.json`, `docs/review/20260301T054050Z/grafana_search.json`, and `docs/review/20260301T054050Z/runtime_routes/internal_obs_http.txt`.
- Score: **52/100**

6) Production hardening (0–100)
- Security baseline: partial. Sensitive mutations now reject unauthenticated callers (`403` on unauthenticated governance deploy and alert ack in `docs/review/20260301T054050Z/runtime_routes/internal_auth_http.txt`), but `/debug/seed-status` remains open and returns live system counts in `docs/review/20260301T054050Z/runtime_routes/internal_basic_http.txt`, matching `cmd/query/main.go:129-135`.
- Secrets/TLS posture: still development-grade. Compose injects local DB credentials and local JWT key paths directly in env vars at `deploy/docker/docker-compose.yml:86`, `deploy/docker/docker-compose.yml:162`, and `deploy/docker/docker-compose.yml:179`.
- Deterministic migrations + recoverability: much better than the prior audit. `migrate` now completes cleanly in `docs/review/20260301T054050Z/migrate.log`, and `008_scores_lineage.sql` is compatible with the `scores` hypertable by avoiding a native PK/FK and using trigger enforcement instead at `sql/migrations/008_scores_lineage.sql:13-56`.
- Recoverability remains imperfect because the product depends on bootstrap data that is not seeded in a clean boot. `users_count=0` in `docs/review/20260301T054050Z/db_users.txt`, so authenticated end-user flows are not actually usable immediately after the clean boot.
- Host networking from this shell is still unreliable: container ports are published in `docs/review/20260301T054050Z/compose_ps.txt`, but direct host curls fail for gateway, query, inference, Prometheus, and Grafana in the captured route artifacts. That weakens local operability and audit reproducibility.
- Rate-limit / CSRF runtime truth is `UNPROVEN` in this pass because no authenticated browser session could be established and no Playwright run was captured.
- Score: **46/100**

Finally:
- Overall % = weighted (UI 25, Quant/ML 25, Data 20, Backend 15, Obs 10, Hardening 5)
- Overall % = **64/100**
- Calculation: `(45*0.25) + (80*0.25) + (74*0.20) + (68*0.15) + (52*0.10) + (46*0.05) = 63.75`

- Top 10 blockers (smallest fix first)
- 1. No users are seeded on a clean boot, so authenticated product flows are not provable; see `docs/review/20260301T054050Z/db_users.txt` and `docs/review/20260301T054050Z/runtime_routes/internal_auth_http.txt`.
- 2. `/debug/seed-status` is publicly readable in the live stack; see `docs/review/20260301T054050Z/runtime_routes/internal_basic_http.txt` and `cmd/query/main.go:129-135`.
- 3. `/sse/alerts` is still mounted outside auth/RBAC; see `cmd/alerts/main.go:118-120`.
- 4. Unauthenticated privileged mutations return `403`, but authn vs authz are still conflated rather than clearly separated into `401` vs `403`; see `docs/review/20260301T054050Z/runtime_routes/internal_auth_http.txt` and `cmd/alerts/main.go:122-125`.
- 5. Go observability is still stubbed at `sentinel_up` only; see `internal/metrics/metrics.go:8-12` and `docs/review/20260301T054050Z/runtime_routes/internal_basic_http.txt`.
- 6. Prometheus has no `http_requests_total` data, so dashboard breadth still overstates backend observability; see `docs/review/20260301T054050Z/runtime_routes/internal_obs_http.txt` and `grafana/dashboards/sentinel-dashboard.json:1-86`.
- 7. Host-port curls fail from this shell even though services are healthy in Docker, which weakens local runtime proof and operator ergonomics; see `docs/review/20260301T054050Z/compose_ps.txt` versus `docs/review/20260301T054050Z/prom_up.json` and the host curl failures in `docs/review/20260301T054050Z/runtime_routes/`.
- 8. Filter state is not URL-canonical, so refresh/shareable UI context is incomplete; see `web/components/AppShell.tsx:23-41`.
- 9. Charts are still single-resolution and `/candles` has no resolution parameter, so “real terminal” depth is incomplete; see `web/components/CandleRiskPanel.tsx:83-86` and `cmd/query/main.go:486-503`.
- 10. Globe interaction is real, but geographic truth is approximate because country matching uses bounding boxes rather than real polygon geometry; see `web/components/RiskGlobe.tsx:7-45` and `web/components/RiskGlobe.tsx:138-165`.

- “What to build next” plan (7 days)
- Day 1: Re-run the same clean-boot audit after proving seeded login users exist on fresh boot, so authenticated API and browser checks move from `UNPROVEN` to runtime-proven.
- Day 2: Capture authenticated `/scores`, `/trust`, `/feed`, and `/command-center` responses in the live stack and compare them against the already-proven DB state.
- Day 3: Add a browser/Playwright evidence pass focused on filter request-param changes, live charts, and globe-to-filter propagation so UI scoring stops relying mainly on source inspection.
- Day 4: Re-run Prometheus queries for real Go request metrics; if they remain empty, observability score should stay capped despite healthy `up` signals.
- Day 5: Re-check unauthenticated surfaces (`/debug/seed-status`, `/sse/alerts`) and update the backend score once those are either intentionally retained or closed.
- Day 6: Reconcile dashboard coverage against actual PromQL non-empty series so Grafana claims match emitted metrics.
- Day 7: Re-run a full forensic pass from `down -v` with the same artifact set and compare deltas only where new runtime proof materially changed confidence.
