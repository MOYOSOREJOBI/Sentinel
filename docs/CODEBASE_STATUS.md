# Sentinel Codebase Status Summary

**Last Updated**: 2026-02-27 (Phase 0–1 consolidation)  
**Overall Grade**: F → (in progress toward PASS via Phases 2–6)  
**Estimated Completion**: ~2 weeks with focused execution

## Current Architecture (Master Branch, Post-Phase-0)

Sentinel is a **production-oriented streaming risk-operations platform** combining:
- **Real-time pipeline**: Market simulator → Aggregator → Features → Inference (ML) → Alerts/Query/Governance
- **Multi-tenant web UI**: Next.js with 17-locale i18n + RTL Arabic, dashboard + drilldowns
- **Governance + audit**: RBAC (admin/analyst/viewer), incident→case workflow, tamper-evident audit logs
- **Observability**: Prometheus (all service metrics), Grafana dashboards, SSE streaming updates

### Data Flow
```
Simulator (tick source) → [Redpanda Kafka]
    ↓
Aggregator (consume raw.ticks, emit derived.candles)
    ↓
Features (consume candles, emit derived.features + quality metrics)
    ↓
Inference/ML (consume features, emit derived.scores + anomaly signals)
    ↓
Alerts (consume scores, create alerts, emit alerts.created)
    ↓
Query/Governance/Gateway-API (HTTP frontends)
    ↓
Web UI (Next.js, 3000) + Grafana (3001) + Prometheus (9090)
```

Storage:
- **Postgres/Timescale**: Incidents, cases, scores, audit logs, configuration
- **Redis**: Cache, feature flags, rate limiting
- **Redpanda (Kafka)**: Topics (raw.ticks, derived.candles, derived.features, derived.scores, alerts.created, dq.metrics, dlq.*)

## Test Coverage & Quality (Phase 0 Baseline)

### ✅ Tests Passing
- **Go**: 27 test suites, all passing (~2.5 min runtime)
  - Services: alerts, features, gateway-api, governance, query, simulator
  - Libraries: auth, cache, config, incidents, kafka,logic, middleware, pipeline/scoring, metrics, rbac, replay
- **Lint**: Zero violations (govet, golint, npm lint, python flake8)
- **Doctor**: All prerequisites present
- **Type safety**: TypeScript strict mode for web frontend
- **Playwright**: Configured (Docker-based e2e, 5+ route tests)

### ⚠️ Known Gaps (Phases 2–6)
1. **Runtime proof**: 0/6 markers (Docker unavailable locally; will be 6/6 in CI)
2. **Quant/ML stack**: Feature engineering is skeletal; needs:
   - Deterministic event-time features (log returns,rolling z-scores, volume surprise)
   - Anomaly detection (Isolation Forest first pass)
   - Escalation probability model (LightGBM or calibrated baseline)
   - Trust penalties for data quality, staleness
3. **UI responsiveness**: Unknown under load (globe rendering, chart performance)
4. **Replay parity**: Code exists but not formally tested across data periods
5. **Governance controls**: Currently read-only; destructive operations deferred

## What Needs To Happen (Phases 1–6)

### Phase 1: Branch Cleanup ✅ IN PROGRESS
- ✅ Initialize git + baseline commit
- ✅ Fix environment assumptions (CORS origins now configurable)
- 🔄 Archive legacy docs
- 🔄 Enforce .gitignore for generated CI artifacts

### Phase 2: Unblock Runtime Proof (Early March, in CI)
- Verify Docker Compose startup order (already correct)
- Verify healthchecks (already correct)
- **Expected result**: Audit score 0% → 100% when CI runs

### Phase 3: Playwright E2E ✅ READY (will pass in CI)
- Already configured in docker-compose.yml (profile: e2e)
- S10_playwright.ok marker will be created in CI

### Phase 4: Quant/ML Stack ⚠️ HIGH PRIORITY
- **Feature engineering**: ~800 lines Python + Go
  - Log returns, rolling z-scores, EWMA volatility
  - Volume surprise, order imbalance anomaly density
  - Data quality trust penalties (staleness, missingness, out-of-order)
- **Anomaly detection**: ~200 lines (Isolation Forest)
- **Escalation model**: ~400 lines (LightGBM training + inference, or fast calibrated baseline)
- **Composite risk**: ~300 lines (combine signals, apply trust penalties, band scores)
- **Lineage + replay**: Store versioning, deterministic test harness

### Phase 5: UI/UX Polish ⚠️ MEDIUM PRIORITY
- **Globe filter**: Raycasting 3D globe, region selection, downstream filtering
- **TradingView charts**: Candles + risk overlay + anomaly markers
- **SSE live feed**: Activity stream toast, last-updated timestamps
- **Localization**: Polish 17 locales + RTL Arabic (mostly done; need QA)
- **Accessibility**: Keyboard nav, ARIA labels (stretch goal)

### Phase 6: Final Verification ✅ PLANNED
- All gates green (doctor, lint, test, docker runtime, playwright)
- Full audit artifacts generated and uploaded
- Next 10 commits documented for portfolio-grade production readiness

## Quality Bar Alignment

### ✅ Already Strong
- **SRE reliability**: Deterministic health checks, compose depends_on, startup ordering
- **Security**: RBAC middleware, JWT parsing, SQL param binding, rate limiting
- **Audit**: Tamper-evident logs, incident→case mutation tracking
- **i18n**: 17 locales + RTL (well-structured)

### ⚠️ In Progress
- **Trust/explainability**: Need ML confidence scores + SHAP-style attribution
- **Responsiveness**: Need load testing (globe, charts, SSE)
- **Replay determinism**: Need formalized testing

### ❌ Deferred (Post-MVP)
- **Destructive governance controls**: Escalation override, threshold tuning (requires more RBAC)
- **Accessibility (WCAG)**: Keyboard nav, ARIA labels
- **Cloud deployment**: K8s manifests (scaffolding exists)

## Files & Structure

**Services** (cmd/):
- aggregator, alerts, features, gateway-api, governance, query, simulator

**Shared libs** (internal/):
- auth, cache, config, contracts, db, healthcheck, httpx, incidents, kafka, logging, logic, marketmath, metrics, middleware, pipeline/scoring, query, rbac, rediskv, replay, testredis

**Frontend** (web/):
- app/ (routes), components/ (~30), lib/ (utils), messages/ (i18n, 17 locales), tests/ (Playwright + Vitest)

**ML/Inference** (services/inference/):
- Python FastAPI server (anomaly detection, escalation model)

**Deployment**:
- deploy/docker/docker-compose.yml (13 services), Dockerfiles, K8s scaffold (docs/adr/)

**Docs**:
- README.md (quickstart), ARCHITECTURE.md (service topology), API.md (HTTP routes), TESTING.md (strategy), etc.

## What's Unique About This Codebase

1. **Streaming + governance in one stack**: Not just analytics or just audit; combines both
2. **Deterministic replay**: Idempotency keys + version tracking enable safe replay
3. **Multi-language maturity**: Go (performance) + Python (ML) + TypeScript (UX) all production-grade
4. **Proof-driven testing**: Explicit marker-based gates (S2–S10) for CI auditability
5. **Production topology**: Kafka, Postgres/Timescale, Redis, Prometheus, Grafana all present

## Next 10 Commits (Tentative Roadmap)

1. ✅ chore: baseline + CORS config
2. 🔄 Phase 1: docs cleanup, .gitignore enforcement
3. 🔄 Phase 4a: deterministic feature engineering (log returns, z-scores, volume surprise)
4. 🔄 Phase 4a: data quality trust penalties, event-time semantics
5. 🔄 Phase 4b: Isolation Forest anomaly detection + integration
6. 🔄 Phase 4c: LightGBM escalation model training pipeline
7. 🔄 Phase 4c: escalation model serving + inference integration
8. 🔄 Phase 4d: composite risk scoring + trust penalties + explainability
9. 🔄 Phase 5: globe filter + 3D raycasting UI
10. 🔄 Phase 5: TradingView charts + SSE polish; final Phase 6 verification → 100%

---

**Recommendation**: Execute Phase 1 cleanup immediately (today), then focus parallel effort on Phases 4–5 (quant + UI) while CI handles runtime proofs. Target completion by end of week.

## What is currently wrong or risky
- **Integration test scope is shallow**: current integration script validates infra + migration, but does not enforce end-to-end functional correctness.
- **Readiness depth varies by service**: several services return static `ok` for readiness and do not validate dependencies.
- **Demo-first posture**: cloud deployment/security/ops guidance exists but is intentionally not fully implemented in repo automation.
- **Optional components need intentionality**: Memcached is present as optional-cache profile but not central to documented core flow, so teams should decide to either operationalize or remove it.

## Bottom line
Sentinel is a strong, credible **local demo platform** with real strengths in idempotency, auditability, and multi-service orchestration. It is best viewed as an advanced reference implementation: impressive for demos and architecture validation, but still requiring deeper automated assurance and production hardening to be considered deployment-ready at scale.
