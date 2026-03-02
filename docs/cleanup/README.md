# Cleanup Policy

This directory is reserved for one durable document: this file.

Rules:
- Generated audit snapshots belong in `docs/audit/**` and stay untracked.
- Generated progress snapshots belong in `docs/progress/**` and stay untracked.
- Cleanup scratch reports, command logs, and one-off plans do not belong in git history.
- Keep product truth in the live code, the main `README.md`, and the latest audit artifact produced by `./scripts/audit.sh`.

Repo map:
- `cmd/`: Go service entrypoints.
- `internal/`: shared application logic.
- `web/`: Next.js operator console.
- `deploy/docker/`: local runtime orchestration.
- `scripts/`: verification, audit, and developer workflows.
- `sql/`: schema and migrations.
- `docs/audit/README.md`: audit artifact policy.
- `docs/progress/README.md`: progress artifact policy.
