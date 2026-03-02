# Merge conflict resolution defaults

When resolving recurring conflicts between long-lived branches, use the following defaults unless a specific change request says otherwise.

## `deploy/docker/docker-compose.yml`

- Keep healthcheck commands as:
  - `test: ["CMD", "/app/service", "healthcheck"]`
- Do **not** switch these to `/service`.
- Do **not** keep duplicate `test:` keys in the same healthcheck block.

Rationale: the runtime binary path in `deploy/docker/Dockerfile-go` is `/app/service`.

## `scripts/gate.sh`

Keep both of these behaviors:

1. Web gate runs i18n checks before build:

```bash
( cd web && npm ci && npm run check:i18n && npm run build )
```

2. Playwright fallback logic:
   - Try local `npx @playwright/test` first.
   - If local run fails and Docker is available, fallback to `./scripts/playwright-docker.sh`.

## `web/package.json`

Keep this script:

```json
"check:i18n": "node scripts/i18n-check.js"
```

It is required by `scripts/gate.sh`.

## Quick verification after conflict resolution

Run:

```bash
rg -n "^(<{7}|={7}|>{7})" -S .
```

Expected result: no conflict markers found.
