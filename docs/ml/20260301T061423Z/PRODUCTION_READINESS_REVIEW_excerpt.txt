# Skeptical Production-Readiness Review (Fact-Based)

## Quant/ML production-mode updates (2026-03-01)
- `scores` persistence is now a real runtime path in `alerts`, and the writer stores `scoring_run_id`, `produced_at`, `artifact_hash`, `model_artifact_hash`, `feature_snapshot_hash`, and `model_version`.
- `/scores` now returns populated lineage fields for each score row, including `model_version`, `produced_at`, and `scoring_run_id`.
- The inference service now loads bundled model artifacts from `services/inference/models/` by default, exposes `fallback_mode`, `model_version`, `model_artifact_hash`, and `trained_at` on `/readyz`, and emits calibration metrics on `/metrics`.
- Model bundle metadata now includes calibration metrics (`auc`, `brier`, `pr_auc`) plus anomaly stability, and `/trust` surfaces that metadata for the UI.
- `integration/ml_scores_persist_test.sh` enforces a clean-boot invariant that `scores.count > 0`, at least one non-fallback model row exists, and lineage columns are non-null before writing `docs/audit/_latest/proofs/S4_ml_real.ok`.

## A. High-level architecture summary
- Current implemented pipeline is effectively:
  - `simulator` produces to `raw.ticks`.
  - `aggregator` consumes `raw.ticks`, writes `raw_ticks` + `candles`, produces `derived.candles`.
  - `features` consumes `derived.candles`, writes `features`, produces `derived.features`.
  - `inference` consumes `derived.features`, produces `derived.scores` only.
  - `alerts` consumes `derived.scores`, writes `scores` + `alerts`, emits SSE + Kafka `alerts.created`.
  - `query` reads from Postgres (`alerts`, `scores`) for frontend.
- Current gating risk: migration `008_scores_lineage.sql` must stay idempotent and compatible with the `scores` hypertable, otherwise the score writer never starts on a clean boot.

## B. Known working paths
- Health endpoints are present for services and return static `ok` for many services.
- Alert creation path appears to work at least superficially:
  - `alerts` consumes `derived.scores` and inserts `alerts` rows on medium/high/critical severities.
  - SSE endpoint (`/sse/alerts`) broadcasts the consumed score payload to clients.
- Login and `/me` work with JWT cookie-based auth in gateway.
- `audit_log` append-only protection is present at DB trigger level.

## C. Concrete risks grouped by severity

### Critical
1. **Scores pipeline clean-boot reliability depends on migration correctness.**
   - Root cause: the score writer is implemented in `alerts`, but a broken migration can prevent the writer from starting.
   - Blast radius: `/scores` remains empty after a fresh boot even though the write path exists in code.
   - User impact: UI suggests model output is absent/inconsistent; trust erosion.
   - Ops impact: impossible to investigate historical score generation on cold start.
   - Fix: keep the lineage migration compatible with the `scores` hypertable and retain the idempotent `alerts` writer.

2. **Privileged and state-changing endpoints lack auth/RBAC enforcement.**
   - Root cause: `alerts` ack and governance routes do not validate JWT/role.
   - Blast radius: any caller can acknowledge alerts, deploy models, start replay.
   - User impact: unauthorized state changes and governance compromise.
   - Ops impact: incident forensics and compliance invalidated.
   - Fix: enforce JWT auth middleware + role checks on alerts/governance service entrypoints.

3. **Startup race on Kafka topic readiness causing dropped/failed production attempts.**
   - Root cause: simulator starts with compose service health, but topic creation runs later (`run-demo` creates topics after services already running).
   - Blast radius: early ticks fail with `UNKNOWN_TOPIC_OR_PARTITION`; data loss at startup window.
   - User impact: intermittent empty/slow data at startup.
   - Ops impact: nondeterministic boot behavior; flaky demos and deployments.
   - Fix: gate producers on topic existence or auto-create with retries/backoff and readiness probe.

### Major
1. **False-negative infrastructure health (memcached).**
   - Root cause: memcached healthcheck uses `nc` not installed in image.
   - Blast radius: stack reports unhealthy even if memcached process itself is fine.
   - User impact: noisy/unreliable operational status.
   - Ops impact: alert fatigue; unclear dependency criticality.
   - Fix: use `memcached-tool`/`bash /dev/tcp`/install netcat explicitly; or remove service if unused.

2. **Metrics contract broken for inference.**
   - Root cause: Prometheus scrapes default `/metrics`; inference exposes only `/healthz` and `/readyz`.
   - Blast radius: inference observability is blind.
   - User impact: failures/perf regressions undetected.
   - Ops impact: cannot track lag, throughput, errors.
   - Fix: add `/metrics` with Prometheus client and instrument Kafka loop.

3. **Audit trail coverage gaps.**
   - Root cause: `alerts/{id}/ack` and governance actions lack authenticated actor identity and some flows not audited with true principal.
   - Blast radius: audit chain exists but may not cover real privileged actor attribution.
   - Fix: require identity and append audited actor for ack/model/replay actions.

4. **Query service lacks request-path observability.**
   - Root cause: only startup/shutdown logs; no request/latency/status logs.
   - Blast radius: production debugging and SLO validation are weak.
   - Fix: middleware for structured request logging + DB query error counters/latency histograms.

### Minor
1. **Timestamp naming inconsistency (`created_at` in DB vs `ts` in API projection).**
   - Not a runtime break in query service (it maps `created_at` -> JSON `ts`), but manual SQL using `alerts.ts` fails, causing operator confusion.
   - Fix: standardize API/DB naming (`created_at` everywhere, or explicit docs/compat alias).

2. **RBAC package exists but is not integrated.**
   - Indicates intended controls are currently bypassed at runtime.

3. **Frontend presents empty scores as benign "No scores yet" without degradation signal.**
   - Distinguishes neither startup lag nor broken score writer.

## D. File-by-file review targets (inspect first)
1. `cmd/alerts/main.go`: score->alert behavior, SSE, ack auth/audit gap.
2. `cmd/query/main.go`: `/alerts` and `/scores` DB reads, timestamp projection, logging gaps.
3. `services/inference/app.py`: `derived.features`->`derived.scores`, missing `/metrics`.
4. `cmd/simulator/main.go` + `scripts/run-demo.sh` + `scripts/create-topics.sh`: startup race and topic readiness.
5. `deploy/docker/docker-compose.yml`: dependency health, memcached probe correctness, startup assumptions.
6. `sql/migrations/001_init.sql` and `002_indexes.sql`: schema truth, timestamp canonical columns, audit append-only trigger.
7. `cmd/gateway-api/main.go` and `internal/auth/auth.go`: cookie/JWT security posture.
8. `cmd/governance/main.go`: privileged operations without auth.
9. `web/app/page.tsx`: handling of empty scores vs active alerts and partial failures.

## E. Contract checks between services
- Topic contract appears aligned by name (`raw.ticks`, `derived.candles`, `derived.features`, `derived.scores`), and the persistence contract is now handled by the `alerts` consumer writing the `scores` table.
- Inference emits payload fields `symbol`, `score`, `severity`, `explanation`, `ts` as unix epoch float; no explicit schema versioning.
- Alerts consumes untyped JSON map; no strict validation beyond `symbol` and `severity`, so producer drift can silently break alerting semantics.
- Query `/alerts` emits `ts` sourced from `created_at`, while DB has no `alerts.ts`; operator docs/queries are drift-prone.
- SSE emits raw score payload from Kafka record, not a stable typed alert envelope.

## F. Database and migration review
- `001_init.sql` creates base schema and append-only audit triggers; currently no `-- +goose Down` section in this file (partial reversibility concern).
- `002_indexes.sql` has Up/Down separation and is reversible.
- Canonical time columns are mixed:
  - `alerts.created_at` (not `ts`)
  - `scores.ts`, `features.ts`, `candles.bucket`
- System still allows some partial-success risk if the stack fails before `alerts` starts, but the runtime path now persists `scores` before creating `alerts`.
- Recommendation: keep persisted `scores` as the prerequisite for alert creation and preserve the current idempotent upsert contract.

## G. Auth / RBAC / audit review
- Gateway login sets cookie with `HttpOnly` but `Secure=false` (acceptable dev default; not prod-safe).
- JWT expiration set to 24h, no refresh token mechanism.
- `auth.Parse` usage has no explicit issuer/audience validation policy.
- Governance endpoints (`/models/deploy`, `/replay/start`) are now auth + RBAC gated.
- Alerts ack endpoint is now auth + RBAC gated and audited.
- Audit chain integrity mechanism is present and good, but privileged route coverage is incomplete.

## H. Frontend contract and UX review
- Frontend polls `/alerts` and `/scores` every 5s and connects SSE.
- Empty `/scores` is rendered as normal empty-state text, which masks broken scoring persistence.
- SSE failure/reconnect/error state is not surfaced to user.
- Alert acknowledge call does not enforce authenticated user in UI flow because backend does not require it.
- No explicit degraded-state banner when alerts are flowing but score history is empty.

## I. Infra / Docker / scripts / observability review
- Compose health check for memcached is objectively broken due missing `nc` binary.
- `run-demo.sh` starts all services before topic creation; simulator can hit non-existent topic partitions early.
- Prometheus scrapes inference target and inference now exposes `/metrics` with scoring latency, fallback mode, and calibration gauges.
- Integration script validates only infra + migration, not end-to-end dataflow.
- Query logging is insufficient for request-level diagnostics.

## J. Test coverage gaps
- `go test -race` passes but almost all packages have no tests.
- Missing highest-value tests:
  1. End-to-end test proving `ticks -> scores persisted -> alerts`.
  2. Contract tests for Kafka payload schemas and field names.
  3. Migration regression tests (including Up/Down safety and rerun idempotency).
  4. Auth/RBAC tests for governance and alert ack permissions.
  5. Observability tests ensuring every service exposes `/metrics` if configured in Prometheus.
  6. Startup-order chaos test: services up before topic creation should self-recover with retries.

## K. Exact improvements to make it more unique
1. Add alert explainability object linking score, feature contributors, and threshold decisions.
2. Add investigation timeline per symbol (raw tick -> candle -> feature -> score -> alert trace IDs).
3. Add drift monitor dashboard: feature distribution drift + model version drift + alert precision proxy.
4. Add replay comparator UI showing baseline vs replayed outcomes with diff summaries.
5. Add operator controls for suppression with mandatory rationale + expiration.

## L. Exact improvements to make it more production-ready
1. Keep the existing `alerts`-hosted score writer healthy on clean boot and continue persisting `derived.scores` into `scores` with idempotency and retries.
2. Enforce auth middleware and RBAC on alerts/governance, plus audit actor attribution.
3. Fix memcached healthcheck or remove memcached if unused.
4. Add inference `/metrics` and align Prometheus scrape paths.
5. Rework startup determinism:
   - create topics before producers start, or
   - producers retry with exponential backoff until metadata ready.
6. Add request/response logging middleware with correlation IDs across all HTTP services.
7. Add CI checks for schema-query alignment and API contract tests.
8. Address critical npm vulnerability in web container and pin patched dependency versions.

## M. Prioritized fix order

### First 1 hour
1. Fix memcached healthcheck command.
2. Add explicit degraded warning in UI when alerts>0 and scores=0.
3. Add request logging middleware to query service.
4. Disable inference target in Prometheus until `/metrics` exists, or add stub `/metrics` immediately.

### First 1 day
1. Implement score persistence consumer/service and wire `/scores` to real data.
2. Add auth/RBAC on governance + alert ack and audit those actions.
3. Fix startup order: topics before simulator start; add produce retry/backoff.
4. Add integration smoke test that fails if scores table remains empty while alerts grow.

### First 1 week
1. Full contract-test suite for Kafka message schemas and API payloads.
2. Production-grade telemetry (metrics, tracing, structured logs, dashboards, alerts).
3. Migration harness with up/down/idempotency checks in CI.
4. Governance hardening (model state machine, approvals, signed deployment records).

## N. Specific code changes to make, with file paths and rationale
1. **Persist scores**
   - Add `cmd/score-writer/main.go` to consume `derived.scores` and `INSERT INTO scores(...) ON CONFLICT DO NOTHING`.
   - Rationale: remove alerts/scores inconsistency and enable historical analysis.

2. **Protect privileged routes**
   - `cmd/alerts/main.go`: require JWT + role for `/alerts/{id}/ack`; append audit with real actor.
   - `cmd/governance/main.go`: require admin role for `/models/deploy`, analyst/admin for `/replay/start`.
   - Rationale: close unauthorized state-change vulnerability.

3. **Fix startup determinism**
   - `scripts/run-demo.sh`: run `create-topics.sh` before starting producer services, or split compose profiles.
   - `internal/kafka/kafka.go`: add retry/backoff helper for produce on unknown-topic metadata failures.

4. **Observability**
   - `services/inference/app.py`: add `/metrics` via `prometheus_client` and counters for consumed/produced/errors.
   - `cmd/query/main.go`: add request logging middleware + per-handler error logs with route context.
   - `prometheus/prometheus.yml`: align scrape path declarations and only scrape endpoints that exist.

5. **Healthcheck correctness**
   - `deploy/docker/docker-compose.yml`: replace memcached probe command with one supported in image, or remove service.
   - Rationale: health reflects reality.

6. **Frontend degraded-state clarity**
   - `web/app/page.tsx`: add explicit warning panel when `alerts.length>0 && scores.length==0` and SSE disconnected states.
   - Rationale: avoid false confidence in model pipeline health.

## O. Contradictions to resolve immediately before trusting the demo
1. **"Core flows pass" vs `scores` table stays empty**: currently consistent with code because score persistence is missing.
2. **"Memcached logs look fine" vs container unhealthy**: healthcheck is broken independently of logs.
3. **Healthz green vs metrics 404**: readiness checks are shallow and do not prove observability readiness.
4. **Alerts API works vs SQL query on `alerts.ts` fails**: canonical DB column is `created_at`; API re-labels it to `ts`.
5. **Tests green vs production confidence low**: most packages are untested; green result is mostly "no tests executed".
