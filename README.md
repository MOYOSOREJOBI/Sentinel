# Sentinel

Sentinel is a streaming risk alerts system built at CalgaryHacks 2026.
It processes ticks into features into scores into alerts.
It supports deterministic replay, governance checks, and an operator console.

## Why it matters

You get a full pipeline that runs live.
Every step is idempotent and logged.
You can replay any incident and prove the data did not change.
You can run in Docker or on Kubernetes.

## Pipeline overview

1. Simulator emits market ticks.
2. Aggregator builds candles and saves them.
3. Features service computes risk features and data‑quality signals.
4. Inference (Python) scores anomaly and escalation.
5. Alerts service turns scores into alerts and incidents.
6. Query service provides read models for the UI.
7. Web console (Next.js) shows command center, queue, cases.

## Features

### Reliability

• Consumers commit offsets only after a successful transaction.
• Replay returns the same outputs for the same input window.
• Circuit suppression stops alerts if noise spikes.

### Governance

• Role‑based access control on mutation endpoints.
• Tamper‑evident audit log with a hash chain.
• CSRF and rate limits are Redis-backed; protected writes fail closed if Redis is unavailable unless `DEV_UNSAFE=true` is set for local debugging.
• Replay jobs persist parity summaries, determinism fingerprints, and exportable replay briefs with provenance.

### Observability

• Every service exposes `/metrics`.
• Grafana dashboards provisioned automatically.
• Runtime‑proof scripts check health, data warmup, dashboards.

### Operator console

• Command center, queue, case workspace
• Trust page and world risk map
• SSE live updates via web proxy routes
• 17 locales including Arabic RTL

## Tech stack

Go services with shared internal libs.
Python inference service.
Kafka/Redpanda, PostgreSQL (Timescale), Redis, Memcached.
Next.js frontend.
Prometheus and Grafana for metrics.
Docker Compose for local dev.
Kubernetes manifests provided.

## Quickstart

Requirements: Docker Desktop.
Optional: Go 1.22, Node 20, Python 3.11.

Bring up the stack:

```bash
make demo
```

This is the supported clean-checkout path. It now:
- generates `keys/` automatically if missing
- boots the stack
- runs migrations
- creates Kafka topics
- seeds demo users and instrument metadata
- auto-switches to the local override ports if the default host ports are already taken
- requires Redis for login throttling and CSRF; if Redis is down, protected writes return `503` unless `DEV_UNSAFE=true` is set for local-only troubleshooting

Demo logins after `make demo`:
- `admin@sentinel.local` / `Sentinel#123`
- `analyst@sentinel.local` / `Sentinel#123`
- `viewer@sentinel.local` / `Sentinel#123`

Or run compose manually:

```bash
make dev-keys
docker compose -f deploy/docker/docker-compose.yml up -d --build
make seed
```

If the default host ports are busy, use the standard local override:

```bash
make up-local
make seed POSTGRES_PORT=55432
```

Equivalent raw compose command:

```bash
docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.override.local.yml up -d --build
```

Open services:

Primary entrypoint: http://localhost:3000
Backend ports (host-side verification only): http://localhost:8080 and http://localhost:8085
Prometheus: http://localhost:9090
Grafana: http://localhost:3001

When the local override is active:
- Web: `http://localhost:33000`
- Gateway API: `http://localhost:58080`
- Query: `http://localhost:58085`
- Alerts: `http://localhost:58083`
- Governance: `http://localhost:58084`
- Aggregator: `http://localhost:58081`
- Features: `http://localhost:58082`
- Inference: `http://localhost:58090`
- Postgres: `localhost:55432`
- Redis: `localhost:56379`
- Redpanda: `localhost:59092`
- Prometheus: `http://localhost:59090`
- Grafana: `http://localhost:33001`

Health and metrics:

```bash
curl -s http://localhost:3000/api/proxy/gateway-api/healthz
curl -s http://localhost:3000/api/proxy/query/metrics | head
```

Data warmup proof:

```bash
curl -s http://localhost:3000/api/proxy/query/debug/seed-status | jq
```

## Development workflow

Run the health check and linting:

```bash
make doctor
make lint
make test
```

To run proofs:

```bash
./scripts/runtime-proof.sh
```

To run Playwright inside Docker:

```bash
./scripts/playwright-docker.sh
```

## Runbook

### How to run

```bash
make demo
```

`make up`, `make up-fast`, `make up-local`, and `make up-fast-local` generate local dev keys automatically before Docker builds. `make demo` and `make demo-local` also seed the demo users automatically.

### Port conflicts

Use the standard local override when the defaults are already occupied:

```bash
make demo-local
```

Or force the demo wrapper to use the override file:

```bash
SENTINEL_USE_LOCAL_OVERRIDE=1 ./scripts/run-demo.sh default
```

To inspect which process owns a conflicting port:

```bash
lsof -nP -iTCP:3000 -sTCP:LISTEN
lsof -nP -iTCP:8080 -sTCP:LISTEN
lsof -nP -iTCP:5432 -sTCP:LISTEN
```

### How to validate

```bash
./scripts/audit.sh
./scripts/prod-readiness.sh
```

### How to demo

1. Log in as `admin@sentinel.local` / `Sentinel#123`.
2. Open Command Center and confirm the demo shows populated country, severity, and histogram charts plus a live market+risk overlay instead of the empty fallback.
3. Open Queue or Feed, then drill into an incident.
4. Use `Ack`, `Escalate`, `Notify on-call`, and `Promote to Case`.
5. Copy or download the incident brief from the incident page.
6. Open Feed to show the Comms panel and the persisted `Next 24h forecast`.
7. Open Grafana to verify `http_requests_total`, latency, and Kafka traffic panels are non-zero.
8. Open Governance, start a replay, then open the replay job to show `MATCH`/`MISMATCH`, determinism status, and export the replay brief.

## Common issues

**UI empty in Docker**

Localhost inside web container refers to itself.
Use server-only upstream env vars:

```
SENTINEL_UPSTREAM_MODE=compose
UPSTREAM_QUERY=http://query:8085
UPSTREAM_GATEWAY_API=http://gateway-api:8080
UPSTREAM_ALERTS=http://alerts:8083
UPSTREAM_GOVERNANCE=http://governance:8084
```

**Auth not redirecting**

`/me` must fail 401 so UI redirects to login.
Do not return viewer on unauthenticated requests.

**Playwright failures**

Use base URL `http://web:3000` inside the Compose network.
Use stable `data-testid` selectors.

## Security notes

Sentinel is a demo platform. Do not deploy it in production without review.
Check secrets management, auth hardening, network policies, rate limits.
See `docs/THREAT_MODEL.md` for details.

## Credits

Built at CalgaryHacks 2026 by a team of four.
Architecture and code led by the repo owner.

Repository: https://github.com/MOYOSOREJOBI/Sentinel.git
