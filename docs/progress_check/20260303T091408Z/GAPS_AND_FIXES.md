# Gaps And Fixes

Ranked by severity first, then user impact, then estimated fix size.

## 1. Fresh compose boot is blocked on a missing repo-root `keys/` directory

Symptom:

- `docker compose -f deploy/docker/docker-compose.yml up -d --build` fails before any containers start.

Root cause:

- `deploy/docker/Dockerfile-go` hard-requires `COPY keys ./keys`, but the documented raw compose path does not generate those dev keys first.

Evidence:

- `deploy/docker/Dockerfile-go:13`
- `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_up.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_ps.txt`

Fix:

- Make the supported `make up` path generate dev keys before compose build so the primary local boot path is self-contained.
- Proposed patch: `docs/progress_check/20260303T091408Z/PATCHES/01_make_up_generates_dev_keys.diff`

Verification:

- `make up`
- `docker compose -f deploy/docker/docker-compose.yml ps`
- `curl -i http://localhost:8080/healthz`

## 2. `scripts/doctor.sh` under-reports missing keys

Symptom:

- Missing JWT key files are printed, but they do not affect the final warning state calculation.

Root cause:

- The two key checks only echo text and never set `warn=1`.

Evidence:

- `scripts/doctor.sh:25-30`
- `docs/progress_check/20260303T091408Z/EVIDENCE/tests/make_doctor.txt`

Fix:

- Count missing key files as warnings so `doctor` cannot present a false-green when only keys are missing.
- Proposed patch: `docs/progress_check/20260303T091408Z/PATCHES/10_doctor_counts_missing_keys.diff`

Verification:

- Remove or rename `keys/` locally.
- `make doctor`
- Confirm the script exits with a warning summary that includes missing keys.

## 3. The README manual compose quickstart is currently false

Symptom:

- The README suggests raw compose boot as a direct alternative, but that path fails on a clean checkout.

Root cause:

- The doc omits the required `make dev-keys` precondition and also omits the user seeding step needed for logins.

Evidence:

- `README.md:72-76`
- `README.md:138-143`
- `scripts/dev-keys.sh:3-10`
- `scripts/run-demo.sh:23-35`
- `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_up.txt`

Fix:

- Update README so the manual path explicitly generates keys first and notes that authenticated demo flows require `make seed`.
- Proposed patch: `docs/progress_check/20260303T091408Z/PATCHES/02_readme_manual_compose_requires_keys_and_seed.diff`

Verification:

- Follow the updated README literally on a clean checkout.
- Confirm the stack builds.
- Confirm `POST /auth/login` succeeds after `make seed`.

## 4. `go test ./...` and `go vet ./...` are broken by the `sentinel/scripts` package layout

Symptom:

- Package-wide Go test and vet fail with `main redeclared in this block`.

Root cause:

- `scripts/backfill.go` and `scripts/seed-users.go` are both `package main` in the same directory, so `./...` treats them as one invalid package.

Evidence:

- `docs/progress_check/20260303T091408Z/EVIDENCE/tests/go_test_count1_gocache.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/tests/go_test_race_gocache.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/tests/go_vet_gocache.txt`

Fix:

- The smallest safe fix is to exclude `sentinel/scripts` from package-wide test and vet targets until the scripts are moved into separate directories.
- Proposed patch: `docs/progress_check/20260303T091408Z/PATCHES/03_make_test_excludes_cli_scripts_package.diff`

Verification:

- `make test`
- `go test ./...` with the updated package list
- `go vet ./...` with the updated package list

## 5. `query` exposes an admin write route without CSRF protection

Symptom:

- `POST /dev/backfill` is authenticated and admin-gated, but there is no CSRF middleware installed on the router.

Root cause:

- `cmd/query/main.go` installs Request ID, CORS, and metrics, but not `RequireCSRFFunc`.

Evidence:

- `cmd/query/main.go:104-148`
- `internal/middleware/csrf.go:61-85`
- `docs/progress_check/20260303T091408Z/ROUTE_SECURITY_MATRIX.md`

Fix:

- Add the same CSRF middleware used by `gateway-api`, `alerts`, and `governance`.
- Proposed patch: `docs/progress_check/20260303T091408Z/PATCHES/04_query_add_csrf_middleware.diff`

Verification:

- Log in as admin.
- Attempt `POST /dev/backfill` without `X-CSRF-Token` and expect `403`.
- Retry with the CSRF token and expect the existing route behavior.

## 6. `POST /auth/logout` is not explicitly authenticated

Symptom:

- The logout handler executes as a public write and clears cookies even when no authenticated subject is present.

Root cause:

- The route has no `RequireAuth` wrapper, and the global CSRF middleware intentionally lets unauthenticated non-login requests pass through when it cannot resolve a subject.

Evidence:

- `cmd/gateway-api/main.go:117-125`
- `cmd/gateway-api/main.go:156-165`
- `internal/middleware/csrf.go:73-80`
- `docs/progress_check/20260303T091408Z/ROUTE_SECURITY_MATRIX.md`

Fix:

- Require a valid authenticated subject before logout proceeds.
- Proposed patch: `docs/progress_check/20260303T091408Z/PATCHES/05_gateway_logout_requires_auth.diff`

Verification:

- `curl -i -X POST http://localhost:8080/auth/logout` without cookies should return `401`.
- Logged-in logout with cookies and a valid `X-CSRF-Token` should still return `204`.

## 7. The Playwright Docker path does not preflight port `5432`

Symptom:

- The Playwright script failed because Docker could not bind host port `5432`.

Root cause:

- `scripts/playwright-docker.sh` jumps straight into `docker compose ... run --rm playwright` without checking for already-allocated host ports.

Evidence:

- `scripts/playwright-docker.sh:3-9`
- `docs/progress_check/20260303T091408Z/EVIDENCE/playwright/results.txt`
- `docs/progress_check/20260303T091408Z/EVIDENCE/playwright/port_5432_listener.txt`

Fix:

- Add an explicit preflight for `5432` (and fail fast with a clear message) before starting compose.
- Proposed patch: `docs/progress_check/20260303T091408Z/PATCHES/06_playwright_docker_preflight_checks.diff`

Verification:

- Run the script while `5432` is in use and confirm it fails immediately with the clearer preflight message.
- Free `5432` and rerun.

## 8. There is no isolated E2E compose override for Playwright

Symptom:

- The Playwright path reuses the same host-port bindings as the normal local stack.

Root cause:

- The repo has only one compose file in active use for both regular local boot and the E2E runner path.

Evidence:

- `deploy/docker/docker-compose.yml:12-270`
- `scripts/playwright-docker.sh:3-9`
- `docs/progress_check/20260303T091408Z/EVIDENCE/playwright/results.txt`

Fix:

- Add an E2E override file that removes host port publication for dependency containers used only inside the compose network.
- Proposed patch: `docs/progress_check/20260303T091408Z/PATCHES/07_add_e2e_compose_override_without_host_ports.diff`

Verification:

- `docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.e2e.yml config`
- Confirm `postgres`, `redis`, and `redpanda` publish no host ports in the merged config.

## 9. The Playwright script does not consume an E2E-specific compose override

Symptom:

- Even if an E2E override exists, the current script only reads a single compose file variable.

Root cause:

- The script builds `DC` from a single `COMPOSE_FILE` string.

Evidence:

- `scripts/playwright-docker.sh:3-8`

Fix:

- Teach the script to layer `deploy/docker/docker-compose.e2e.yml` when it exists.
- Proposed patch: `docs/progress_check/20260303T091408Z/PATCHES/08_playwright_docker_use_e2e_override.diff`

Verification:

- `./scripts/playwright-docker.sh`
- Confirm the generated `docker compose` command uses both files.
- Confirm E2E services start without host-port collisions from the dependency stack.

## 10. `docs/PRODUCTION_READINESS_REVIEW.md` is stale and now contradicts the code

Symptom:

- The review still claims auth is missing, `run-demo.sh` creates topics too late, and inference lacks `/metrics`, all of which are no longer true in current source.

Root cause:

- The doc was not updated after the code changed.

Evidence:

- `docs/PRODUCTION_READINESS_REVIEW.md:38-65`
- `docs/PRODUCTION_READINESS_REVIEW.md:115-145`
- `cmd/alerts/main.go:116-119`
- `cmd/governance/main.go:79-81`
- `scripts/run-demo.sh:30-33`
- `services/inference/app.py:274-276`

Fix:

- Replace the stale claims with current, narrower statements that match the source.
- Proposed patch: `docs/progress_check/20260303T091408Z/PATCHES/09_production_readiness_review_stale_claims.diff`

Verification:

- Re-run this audit section by section.
- Confirm every updated statement can be traced to current code or current runtime evidence.

## Residual unknowns

These remain `UNKNOWN` until the fresh compose boot is fixed:

- Live DB counts and joins
- Live auth and CSRF responses
- Live Prometheus and Grafana health
- Live UI payloads through the Next proxy
