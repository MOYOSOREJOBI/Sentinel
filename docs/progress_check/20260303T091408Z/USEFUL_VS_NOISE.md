# Useful Vs Noise

## Core product code

These directories are core product code and should stay in the active engineering surface.

- `cmd/`
- `internal/`
- `services/inference/`
- `web/`
- `deploy/`
- `sql/`
- `prometheus/`
- `grafana/`
- `scripts/`
- `integration/`

These are the runtime and build surfaces referenced by the current audit evidence:

- Service binaries and routers: `cmd/*`
- Shared behavior and persistence: `internal/*`
- Python inference runtime: `services/inference/*`
- UI and proxy routes: `web/*`
- Local runtime boot path: `deploy/docker/*`

## Generated or audit artifacts

These are useful as evidence or historical breadcrumbs, but they are not product source.

- `docs/progress_check/*`
- `docs/review/*`
- `docs/ml/*`
- `docs/ui_perf/*`

Recommendation:

- Keep them for traceability.
- Add retention rules or archive older runs outside the main repo if disk usage matters.

## Reference-only content

These paths look like reference material or imported experiments, not active Sentinel source.

- `docs/Asset-Manager/`

Evidence:

- It contains a separate `package.json`, separate client/server trees, replit agent state files, and attached prompt material.
- It is one of the largest source-adjacent folders by size in the repo.
- `docs/progress_check/20260303T091408Z/INVENTORY/top_size_offenders.txt`

Recommendation:

- Treat it as reference-only.
- Move it behind a clearer archive/reference convention or ignore it in active code search paths.

## Rebuildable artifacts

These are not source of truth and can be regenerated.

- `alerts` (compiled binary at repo root)
- `query` (compiled binary at repo root)

Evidence:

- `docs/progress_check/20260303T091408Z/INVENTORY/top_size_offenders.txt`

Recommendation:

- Add them to `.gitignore` if they are not intentionally versioned deliverables.
- If they must stay, document why checked-in binaries are required.

## Bloat and repo drag

These are the main size offenders that add little product value during routine review.

- `.git/objects/pack/*` (expected VCS storage, but large)
- Historical audit folders under `docs/progress_check/20260301T165141Z`
- Historical review folders under `docs/review/*`
- Historical ML evidence under `docs/ml/*`
- `docs/Asset-Manager/.local/state/replit/*`

Evidence:

- `docs/progress_check/20260303T091408Z/INVENTORY/top_size_offenders.txt`

Recommendation:

- Do not delete blindly.
- Archive old evidence runs.
- Ignore clearly generated local-state subtrees such as `docs/Asset-Manager/.local/` if they are not intentionally versioned.
