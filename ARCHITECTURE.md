# Sentinel Architecture

Sentinel is a streaming risk pipeline: simulator -> aggregator -> features -> inference -> alerts/query/governance.

## Phase 2 hardening
- Deterministic startup gates in Compose (`migrate`, `topics-init`).
- Dependency-aware readiness checks in Go services and inference.
- Security hardening in JWT parsing and RBAC negative tests.
- Expanded automated tests across frontend, Go, Python, and integration harness.
