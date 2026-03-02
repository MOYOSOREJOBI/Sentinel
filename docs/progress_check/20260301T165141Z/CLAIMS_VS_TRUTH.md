# Claims vs Truth

Scope: compare repo-facing claims (README and local docs) against current code and this audit’s runtime evidence.

## README Claims

| Claim | Source | Truth | Evidence |
| --- | --- | --- | --- |
| “Every step is idempotent and logged.” | `README.md:9-12` | Partial. `alerts` uses score upsert by idempotency key, but `features` still writes DB then publishes Kafka without a shared transaction | `cmd/alerts/main.go:455-470`; `cmd/features/main.go:175-181` |
| “Consumers commit offsets only after a successful transaction.” | `README.md:26-30` | Overstated. The code shows some ordered write-then-publish behavior, but this audit did not prove transactional offset commit semantics across the full pipeline | `UNKNOWN`; minimum proof missing: consumer code path + offset commit behavior + fault-injection log |
| “Role-based access control on mutation endpoints.” | `README.md:32-36` | True for tested paths in this run | `cmd/alerts/main.go:116-133`; `cmd/governance/main.go:78-80`, `cmd/governance/main.go:127-153`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/alerts_ack_unauth.txt`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/alerts_ack_viewer.txt` |
| “CSRF and rate limits with Redis state when configured.” | `README.md:34-36` | Partial. Middleware exists, but logout succeeded without CSRF header and rate limiting did not trigger in the login probe | `internal/middleware/csrf.go:61-84`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/gateway_logout_no_csrf.txt`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/login_rate_limit_codes.txt` |
| “Every service exposes /metrics.” | `README.md:38-42` | False | `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_metrics.txt`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/simulator_metrics.txt` |
| “SSE live updates via web proxy routes.” | `README.md:44-49` | Partial. The SSE route exists, but current runtime emits degraded events because upstream auth fails through the proxy | `docs/progress_check/20260301T165141Z/EVIDENCE/curl/sse_queue.txt` |
| “Log in as admin… Open Queue or Feed…” | `README.md:136-143` | Not currently true in browser proof. Playwright login did not leave `/login` | `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt` |
| “UI empty in Docker… use compose upstream env vars” | `README.md:147-158` | The underlying statement is true: inside `web`, `query:8085` works and `localhost:8085` fails | `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_container_query_dns_wget.txt`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_container_query_localhost_wget.txt` |

## `docs/PRODUCTION_READINESS_REVIEW.md` Claims

This file is internally contradictory.

### Claims that match current reality

- Quant/ML remediation claims at `docs/PRODUCTION_READINESS_REVIEW.md:3-8` are consistent with current source and runtime:
  - score persistence
  - lineage fields
  - non-fallback model bundle
  - trust/calibration metadata

Supporting proof:

- `cmd/alerts/main.go:467-470`
- `services/inference/app.py:204-236`
- `services/inference/app.py:256-270`
- `internal/query/trust.go:56-79`
- `docs/progress_check/20260301T165141Z/EVIDENCE/metrics/inference_readyz.json`

### Claims that are stale or false now

| Claim | Source | Why False / Stale | Current Evidence |
| --- | --- | --- | --- |
| “Privileged and state-changing endpoints lack auth/RBAC enforcement.” | `docs/PRODUCTION_READINESS_REVIEW.md:38-43` | No longer true in current code/runtime for the tested routes | `cmd/alerts/main.go:116-133`; `cmd/governance/main.go:78-80`, `cmd/governance/main.go:127-153`; security curl proofs |
| “Metrics contract broken for inference… inference exposes only /healthz and /readyz.” | `docs/PRODUCTION_READINESS_REVIEW.md:60-65` | No longer true; inference exposes `/metrics` and emits scoring + bundle metrics | `services/inference/app.py:37-51`; `services/inference/app.py:246-270`; `docs/progress_check/20260301T165141Z/EVIDENCE/metrics/inference_readyz.json` |
| “RBAC package exists but is not integrated.” | `docs/PRODUCTION_READINESS_REVIEW.md:82-83` | No longer true | `internal/authz/middleware.go:56-85`; route wiring in `cmd/alerts/main.go:114-120` and `cmd/governance/main.go:78-81` |
| “Alert acknowledge call does not enforce authenticated user in UI flow because backend does not require it.” | `docs/PRODUCTION_READINESS_REVIEW.md:123-128` | Backend now requires auth/RBAC; the current issue is web-proxy auth failure, not missing backend auth | `cmd/alerts/main.go:116-133`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/alerts_ack_unauth.txt`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/alerts_ack_viewer.txt` |

## “Bloomberg / Twitter / Goldman” Style Claims

There is no single local “resume bullets” file in this audit scope. This section evaluates the *shape* of those implied product claims against runtime truth.

| Implied Claim | Truth | Evidence |
| --- | --- | --- |
| “Bloomberg-density operator console” | Partial. The UI has dense nav, filters, chart, trust, globe, and feed components, but the current web-proxy login path blocks the primary end-user flow | `web/components/AppShell.tsx:12-181`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_proxy_login_raw.txt`; `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt` |
| “Twitter-like realtime feed” | Partial. Feed exists and SSE exists, but current SSE proof is degraded because upstream auth fails | `docs/progress_check/20260301T165141Z/INVENTORY/ui_refs.txt`; `docs/progress_check/20260301T165141Z/EVIDENCE/curl/sse_queue.txt` |
| “Goldman-grade risk engine” | Overclaimed if stated that strongly. The ML/Quant pipeline is real and lineage-aware, but production confidence is limited by low integration coverage and missing backtest output in this audit pass | `services/inference/app.py:183-243`; `internal/query/trust.go:65-79`; `docs/progress_check/20260301T165141Z/EVIDENCE/tests/cover_func.txt` |

## Lie Points

1. The repo’s own audit says `100% PASS`, but the traceability file still carries stale skip notes.
   - `docs/audit/20260301T163152Z/RESULT.md`
   - `docs/audit/20260301T163152Z/TRACEABILITY.md`
2. README says every service exposes `/metrics`, but `web /metrics` is `404`.
   - `README.md:38-42`
   - `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_metrics.txt`
3. README demo implies browser login works, but the current E2E proof shows it does not.
   - `README.md:136-143`
   - `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt`
4. The UI surface implies realtime via proxy, but the current runtime proof shows degraded SSE due to proxy auth failure.
   - `README.md:44-49`
   - `docs/progress_check/20260301T165141Z/EVIDENCE/curl/sse_queue.txt`
