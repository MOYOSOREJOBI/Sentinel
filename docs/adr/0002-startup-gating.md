# ADR 0002: Startup gating with one-shot jobs

## Decision
Use Compose one-shot services (`migrate`, `topics-init`) and `depends_on: service_completed_successfully` for deterministic startup.

## Consequences
Eliminates race where workers process before schema/topics exist.
