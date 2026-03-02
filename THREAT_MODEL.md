# Threat Model

## Key risks
- Unauthorized admin actions (model deploy, replay start).
- Token forgery/algorithm confusion.
- Data tampering in audit log.

## Mitigations
- Server-side RBAC enforcement.
- JWT parser enforces RS256 and required claims.
- Append-only audit log with hash-chain verification.
