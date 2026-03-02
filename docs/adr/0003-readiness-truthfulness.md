# ADR 0003: Readiness must reflect dependency truth

## Decision
`/readyz` returns non-200 until required dependencies (DB/Kafka/model load) are actually ready.

## Consequences
Safer orchestration behavior and reduced false-positive health.
