# Sentinel Runbook

## Bring up stack

Recommended (full local bootstrap):

```bash
./scripts/local-bootstrap.sh ~/Desktop/fast
# optional: MODE=fast ./scripts/local-bootstrap.sh ~/Desktop/fast
```

Manual compose flow:

```bash
docker compose -f deploy/docker/docker-compose.yml up -d --build
```

## Verify readiness
- gateway: `/readyz`
- query: `/readyz`
- alerts/features/aggregator/governance: `/readyz`
- inference: `/readyz`

## Audit chain verification
Call `GET /audit/verify` as admin via gateway.

## Failure triage
1. Check `docker compose ps` and healthchecks.
2. Check migration and topics-init jobs exited successfully.
3. Check service logs for dependency failures.
4. Quick log bundle:

```bash
docker compose -f deploy/docker/docker-compose.yml logs postgres --tail=200
docker compose -f deploy/docker/docker-compose.yml logs query --tail=120
docker compose -f deploy/docker/docker-compose.yml logs web --tail=120
```
