# Gaps and Fixes

This file is ranked by smallest high-leverage fix first. Each item is evidence-backed and includes the minimal verification command or artifact to prove the fix.

## 1. Web proxy login returns 401

- Severity: Critical
- Why it matters: this blocks the main end-user path and causes proxy-backed UI data fetches and SSE to degrade.
- Evidence:
  - `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_proxy_login_raw.txt`
  - `docs/progress_check/20260301T165141Z/EVIDENCE/curl/sse_queue.txt`
  - `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt`
- Code to inspect:
  - `web/app/api/proxy/[service]/[...path]/route.ts:26-39`
  - `cmd/gateway-api/main.go:78-113`
- Minimal fix plan:
  - Trace the proxied login request as it leaves Next and reaches `gateway-api`.
  - Confirm request body and headers are preserved exactly on `POST /auth/login`.
  - Reconcile the current proxy route with the actual working gateway login contract.
- Proof target:
  - `curl -i -sS -X POST http://localhost:3000/api/proxy/gateway-api/auth/login ...` returns `200`.
  - Playwright login transitions from `/login` to `/command-center`.

## 2. CSRF is not enforced on authenticated logout

- Severity: Critical
- Why it matters: this directly contradicts the current CSRF claim.
- Evidence:
  - `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/gateway_logout_no_csrf.txt`
- Code to inspect:
  - `internal/middleware/csrf.go:73-82`
  - `cmd/gateway-api/main.go:115-123`
- Minimal fix plan:
  - Decide whether logout should require CSRF for authenticated callers.
  - If yes, stop bypassing CSRF for authenticated requests that lack a token.
- Proof target:
  - Re-run the same logout curl without `X-CSRF-Token`; expect `403`.

## 3. Playwright login E2E is failing

- Severity: Critical
- Why it matters: the repo’s browser-level proof currently disagrees with the README demo claim.
- Evidence:
  - `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt`
- Code to inspect:
  - `web/e2e/sentinel.spec.ts`
  - Login page component and `requireAuth` client flow
- Minimal fix plan:
  - Fix the underlying login path first; then rerun the same Playwright flow.
  - Persist traces/screenshots to the host so failures are inspectable outside the container.
- Proof target:
  - `docs/progress_check/.../EVIDENCE/playwright/results.txt` shows login spec green.

## 4. Rate limit could not be proven

- Severity: Major
- Why it matters: the middleware exists, but the runtime check did not observe a single `429`.
- Evidence:
  - `docs/progress_check/20260301T165141Z/EVIDENCE/curl/security/login_rate_limit_codes.txt`
- Code to inspect:
  - `cmd/gateway-api/main.go:78-89`
  - `internal/middleware/ratelimit.go`
- Minimal fix plan:
  - Confirm the probe is reusing the same subject key and not accidentally varying the rate-limit key.
  - If configured limits are too high for the probe, adjust the proof script, not necessarily the app logic.
- Proof target:
  - Capture a login burst that returns at least one `429`.

## 5. `features` stage still is not atomic across DB + Kafka

- Severity: Major
- Why it matters: DB insert and Kafka publish are separate operations, so partial success is possible.
- Evidence:
  - `cmd/features/main.go:175-181`
- Minimal fix plan:
  - Add a failure-mode test around the current insert-then-publish path.
  - If partial success is observed, introduce an outbox or explicit retry/reconcile strategy.
- Proof target:
  - Fault injection shows no silent divergence between `features` table and `derived.features` topic.

## 6. Go coverage is too low on critical orchestration paths

- Severity: Major
- Why it matters: production confidence is low even though unit tests are passing.
- Evidence:
  - `docs/progress_check/20260301T165141Z/EVIDENCE/tests/cover_func.txt`
- Specific low-coverage hotspots:
  - `cmd/alerts/main.go:42`, `cmd/alerts/main.go:395`, `cmd/alerts/main.go:621`
  - `cmd/governance/main.go:33`, `cmd/governance/main.go:385`
  - `cmd/query/main.go:80`, `cmd/query/main.go:601`
  - `internal/incidents/incidents.go:14-101`
  - `internal/replay/runner.go`
- Minimal fix plan:
  - Add targeted handler and pipeline tests for exactly these functions first.
- Proof target:
  - `go tool cover -func` shows these hotspots materially above `0.0%`.

## 7. README overclaims metrics coverage

- Severity: Major
- Why it matters: operator-facing docs are overstating live observability.
- Evidence:
  - `README.md:38-42`
  - `docs/progress_check/20260301T165141Z/EVIDENCE/curl/web_metrics.txt`
  - `docs/progress_check/20260301T165141Z/EVIDENCE/curl/simulator_metrics.txt`
- Minimal fix plan:
  - Either make the missing endpoints real, or narrow the claim to “core backend services”.
- Proof target:
  - README and runtime agree.

## 8. Current UI proofs are partially blind because Playwright artifacts were not persisted

- Severity: Major
- Why it matters: failing browser runs are harder to debug and harder to audit.
- Evidence:
  - `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/results.txt`
  - `docs/progress_check/20260301T165141Z/EVIDENCE/playwright/artifacts.txt`
- Minimal fix plan:
  - Change the Playwright-in-Docker script to copy `test-results/` back into the workspace.
- Proof target:
  - Host-side traces/screenshots exist under `docs/progress_check/.../EVIDENCE/playwright/`.

## 9. `docs/PRODUCTION_READINESS_REVIEW.md` is stale in important sections

- Severity: Minor
- Why it matters: it now undermines trust because it mixes fixed issues with outdated warnings.
- Evidence:
  - `docs/PRODUCTION_READINESS_REVIEW.md:38-66`
  - `docs/PRODUCTION_READINESS_REVIEW.md:82-85`
  - `docs/PRODUCTION_READINESS_REVIEW.md:123-128`
- Minimal fix plan:
  - Split “historical findings” from “current findings” or archive the stale portions.
- Proof target:
  - The doc no longer contradicts the current source/runtime on auth, inference metrics, and RBAC.

## 10. Reference asset ballast is inflating repo noise

- Severity: Minor
- Why it matters: `docs/Asset-Manager` includes an embedded `.git` directory and reference history that should not affect production code.
- Evidence:
  - `docs/progress_check/20260301T165141Z/INVENTORY/asset_manager_files.txt`
- Minimal fix plan:
  - Treat it as archived reference material or vendor content, not active source.
- Proof target:
  - Inventory shows only the intended reference assets remain.
