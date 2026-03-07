# Route Security Matrix

Matrix built from:

- `docs/progress_check/20260303T091408Z/INVENTORY/route_inventory.txt`
- `cmd/*/main.go`
- `services/inference/app.py`
- `web/app/api/*/route.ts`

Legend:

- `public read`
- `authenticated read`
- `public write`
- `authenticated write`
- `admin-only write`
- `UNKNOWN`

## Aggregator

| Route | Method | Class | Auth | RBAC | CSRF | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| `/` | GET | public read | none | none | n/a | `cmd/aggregator/main.go:63-68` |
| `/healthz` | GET | public read | none | none | n/a | `cmd/aggregator/main.go:63-64` |
| `/readyz` | GET | public read | none | none | n/a | `cmd/aggregator/main.go:69-85` |
| `/metrics` | GET | public read | none | none | n/a | `cmd/aggregator/main.go:86` |

## Features

| Route | Method | Class | Auth | RBAC | CSRF | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| `/` | GET | public read | none | none | n/a | `cmd/features/main.go:100-105` |
| `/healthz` | GET | public read | none | none | n/a | `cmd/features/main.go:100-101` |
| `/readyz` | GET | public read | none | none | n/a | `cmd/features/main.go:106-122` |
| `/metrics` | GET | public read | none | none | n/a | `cmd/features/main.go:123` |

## Simulator

| Route | Method | Class | Auth | RBAC | CSRF | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| `/` | GET | public read | none | none | n/a | `cmd/simulator/main.go:61-66` |
| `/healthz` | GET | public read | none | none | n/a | `cmd/simulator/main.go:61-62` |
| `/readyz` | GET | public read | none | none | n/a | `cmd/simulator/main.go:67-75` |
| `/metrics` | GET | public read | none | none | n/a | `cmd/simulator/main.go:76` |

## Inference

| Route | Method | Class | Auth | RBAC | CSRF | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| `/` | GET | public read | none | none | n/a | `services/inference/app.py:246-248` |
| `/healthz` | GET | public read | none | none | n/a | `services/inference/app.py:251-253` |
| `/readyz` | GET | public read | none | none | n/a | `services/inference/app.py:256-271` |
| `/metrics` | GET | public read | none | none | n/a | `services/inference/app.py:274-276` |

## Gateway API

Global security wiring:

- Conditional CSRF middleware is installed for all non-GET routes: `cmd/gateway-api/main.go:52-58`
- CSRF behavior: login is exempt; if no authenticated subject is present, the request is allowed through without CSRF rejection; otherwise `X-CSRF-Token` is required: `internal/middleware/csrf.go:61-85`

| Route | Method | Class | Auth enforcement | RBAC enforcement | CSRF enforcement | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| `/` | GET | public read | none | none | n/a | `cmd/gateway-api/main.go:60-70` |
| `/healthz` | GET | public read | none | none | n/a | `cmd/gateway-api/main.go:59` |
| `/readyz` | GET | public read | none | none | n/a | `cmd/gateway-api/main.go:72-78` |
| `/metrics` | GET | public read | none | none | n/a | `cmd/gateway-api/main.go:79` |
| `/auth/login` | POST | public write | none | none | middleware explicitly exempts `/auth/login` | `cmd/gateway-api/main.go:81-115`; `internal/middleware/csrf.go:68-71` |
| `/auth/logout` | POST | public write | no required auth; handler checks auth only to clear bound CSRF state | none | middleware enforces CSRF only when a valid subject can be extracted; unauthenticated requests pass through | `cmd/gateway-api/main.go:117-125`; `cmd/gateway-api/main.go:156-165`; `internal/middleware/csrf.go:73-80` |
| `/me` | GET | authenticated read | inline `authn(...)` inside handler | role returned, not enforced | n/a | `cmd/gateway-api/main.go:128-135` |

## Query

Global security wiring:

- Request ID, CORS, and metrics are installed.
- No CSRF middleware is installed on the router.
- Evidence: `cmd/query/main.go:104-128`

| Route | Method | Class | Auth enforcement | RBAC enforcement | CSRF enforcement | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| `/` | GET | public read | none | none | n/a | `cmd/query/main.go:109-120` |
| `/healthz` | GET | public read | none | none | n/a | `cmd/query/main.go:108` |
| `/readyz` | GET | public read | none | none | n/a | `cmd/query/main.go:121-127` |
| `/metrics` | GET | public read | none | none | n/a | `cmd/query/main.go:128` |
| `/debug/seed-status` | GET | authenticated read | `authz.RequirePerm(pub, "read")` middleware wrapper | same middleware | none | `cmd/query/main.go:136`; `internal/authz/middleware.go:69-85` |
| `/dev/backfill` | POST | admin-only write | `authz.RequirePerm(pub, "admin")` middleware wrapper | same middleware | none present in router | `cmd/query/main.go:137-148`; `internal/authz/middleware.go:69-85` |
| `/queue` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:150-169` |
| `/command-center` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:150-186` |
| `/feed` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:150-230` |
| `/search-suggest` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:150-330` |
| `/trust` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:331-346` |
| `/world-map` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:347-363` |
| `/executive-summary` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:364-425` |
| `/governance/summary` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:426-453` |
| `/replay/{job}` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:454-476` |
| `/incident/{id}` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:477-490` |
| `/incident/{id}/brief` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:491-518` |
| `/cases` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:519-536` |
| `/case/{id}` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:537-548` |
| `/scores` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:549-570` |
| `/candles` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:571-594` |
| `/forecast` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:595-611` |
| `/stream/queue` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:612-619` |
| `/stream/command-center` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:620-623` |
| `/stream/trust` | GET | authenticated read | group middleware `pr.Use(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/query/main.go:624-627` |

## Alerts

Global security wiring:

- Global CSRF middleware: `cmd/alerts/main.go:81`
- Write group: rate limit + `RequireAnyPerm(pub, "alerts:write", "*")`: `cmd/alerts/main.go:116-119`
- Read group: `RequirePerm(pub, "read")`: `cmd/alerts/main.go:120`

| Route | Method | Class | Auth enforcement | RBAC enforcement | CSRF enforcement | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| `/` | GET | public read | none | none | n/a | `cmd/alerts/main.go:83-94` |
| `/healthz` | GET | public read | none | none | n/a | `cmd/alerts/main.go:82` |
| `/readyz` | GET | public read | none | none | n/a | `cmd/alerts/main.go:95-111` |
| `/metrics` | GET | public read | none | none | n/a | `cmd/alerts/main.go:113` |
| `/sse/alerts` | GET | authenticated read | `authz.RequirePerm(pub, "read")` wrapper | same middleware | n/a | `cmd/alerts/main.go:114`; `internal/authz/middleware.go:84-85` |
| `/alerts/{id}/ack` | POST | authenticated write | mutation group uses `RequireAnyPerm(...)` | mutation group permits `alerts:write` or `*` | global CSRF middleware | `cmd/alerts/main.go:116-123`; `cmd/alerts/main.go:122-134`; `internal/authz/middleware.go:69-85`; `internal/middleware/csrf.go:61-85` |
| `/incidents/{id}/transition` | POST | authenticated write | mutation group uses `RequireAnyPerm(...)` | mutation group permits `alerts:write` or `*` | global CSRF middleware | `cmd/alerts/main.go:116-119`; `cmd/alerts/main.go:136-172` |
| `/cases` | POST | authenticated write | mutation group uses `RequireAnyPerm(...)` | mutation group permits `alerts:write` or `*` | global CSRF middleware | `cmd/alerts/main.go:116-119`; `cmd/alerts/main.go:174-185` |
| `/incidents/{id}/promote-case` | POST | authenticated write | mutation group uses `RequireAnyPerm(...)` | mutation group permits `alerts:write` or `*` | global CSRF middleware | `cmd/alerts/main.go:116-119`; `cmd/alerts/main.go:187-202` |
| `/incidents/{id}/notify-oncall` | POST | authenticated write | mutation group uses `RequireAnyPerm(...)` | mutation group permits `alerts:write` or `*` | global CSRF middleware | `cmd/alerts/main.go:116-119`; `cmd/alerts/main.go:203-217` |
| `/cases` | GET | authenticated read | `read := r.With(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/alerts/main.go:120`; `cmd/alerts/main.go:219-236` |
| `/notifications` | GET | authenticated read | `read := r.With(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/alerts/main.go:120`; `cmd/alerts/main.go:237-255` |
| `/cases/{id}` | GET | authenticated read | `read := r.With(authz.RequirePerm(pub, "read"))` | same middleware | n/a | `cmd/alerts/main.go:120`; `cmd/alerts/main.go:257-279` |
| `/cases/{id}` | PATCH | authenticated write | mutation group uses `RequireAnyPerm(...)` | mutation group permits `alerts:write` or `*` | global CSRF middleware | `cmd/alerts/main.go:116-119`; `cmd/alerts/main.go:281-320` |
| `/cases/{id}/notes` | POST | authenticated write | mutation group uses `RequireAnyPerm(...)` | mutation group permits `alerts:write` or `*` | global CSRF middleware | `cmd/alerts/main.go:116-119`; `cmd/alerts/main.go:322-342` |
| `/cases/{id}/evidence` | POST | authenticated write | mutation group uses `RequireAnyPerm(...)` | mutation group permits `alerts:write` or `*` | global CSRF middleware | `cmd/alerts/main.go:116-119`; `cmd/alerts/main.go:344-366` |
| `/cases/{id}/disposition` | POST | authenticated write | mutation group uses `RequireAnyPerm(...)` | mutation group permits `alerts:write` or `*` | global CSRF middleware | `cmd/alerts/main.go:116-119`; `cmd/alerts/main.go:368-390` |

## Governance

Global security wiring:

- Global CSRF middleware: `cmd/governance/main.go:63`
- Read group: `RequirePerm(pub, "read")`: `cmd/governance/main.go:78`
- Admin write group: `RequireAnyPerm(pub, "*", "model:deploy")`: `cmd/governance/main.go:79`
- Workflow write group: `RequireAnyPerm(pub, "replay:write", "*")`: `cmd/governance/main.go:80`
- Policy write group: `RequireAnyPerm(pub, "*")`: `cmd/governance/main.go:81`

| Route | Method | Class | Auth enforcement | RBAC enforcement | CSRF enforcement | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| `/` | GET | public read | none | none | n/a | `cmd/governance/main.go:65-76` |
| `/healthz` | GET | public read | none | none | n/a | `cmd/governance/main.go:64` |
| `/readyz` | GET | public read | none | none | n/a | `cmd/governance/main.go:77` |
| `/metrics` | GET | public read | none | none | n/a | `cmd/governance/main.go:123` |
| `/governance/summary` | GET | authenticated read | read group `RequirePerm(pub, "read")` | same middleware | n/a | `cmd/governance/main.go:78`; `cmd/governance/main.go:83-91` |
| `/active-models` | GET | authenticated read | read group `RequirePerm(pub, "read")` | same middleware | n/a | `cmd/governance/main.go:78`; `cmd/governance/main.go:93-121` |
| `/audit/verify` | GET | authenticated read | read group `RequirePerm(pub, "read")` | same middleware | n/a | `cmd/governance/main.go:78`; `cmd/governance/main.go:124-136` |
| `/models/deploy` | POST | admin-only write | admin group uses `RequireAnyPerm(pub, "*", "model:deploy")` | same middleware | global CSRF middleware | `cmd/governance/main.go:79`; `cmd/governance/main.go:137-163`; `internal/authz/middleware.go:69-85`; `internal/middleware/csrf.go:61-85` |
| `/replay/start` | POST | authenticated write | workflow group uses `RequireAnyPerm(pub, "replay:write", "*")` | same middleware | global CSRF middleware | `cmd/governance/main.go:80`; `cmd/governance/main.go:165-205` |
| `/models` | GET | authenticated read | read group `RequirePerm(pub, "read")` | same middleware | n/a | `cmd/governance/main.go:78`; `cmd/governance/main.go:207-209` |
| `/models/deployments` | GET | authenticated read | read group `RequirePerm(pub, "read")` | same middleware | n/a | `cmd/governance/main.go:78`; `cmd/governance/main.go:211-213` |
| `/replay/jobs` | GET | authenticated read | read group `RequirePerm(pub, "read")` | same middleware | n/a | `cmd/governance/main.go:78`; `cmd/governance/main.go:215-217` |
| `/escalation-policies` | GET | authenticated read | read group `RequirePerm(pub, "read")` | same middleware | n/a | `cmd/governance/main.go:78`; `cmd/governance/main.go:219-221` |
| `/escalation-policies` | POST | admin-only write | policy group uses `RequireAnyPerm(pub, "*")` | effectively admin-only because only `admin` has `*` in current RBAC map | global CSRF middleware | `cmd/governance/main.go:81`; `cmd/governance/main.go:223-243`; `internal/rbac/rbac.go:8-18` |
| `/escalation-policies` | PUT | admin-only write | policy group uses `RequireAnyPerm(pub, "*")` | effectively admin-only because only `admin` has `*` in current RBAC map | global CSRF middleware | `cmd/governance/main.go:81`; `cmd/governance/main.go:245-282`; `internal/rbac/rbac.go:8-18` |
| `/escalation-policies/{id}` | DELETE | admin-only write | policy group uses `RequireAnyPerm(pub, "*")` | effectively admin-only because only `admin` has `*` in current RBAC map | global CSRF middleware | `cmd/governance/main.go:81`; `cmd/governance/main.go:284-302`; `internal/rbac/rbac.go:8-18` |
| `/on-call` | GET | authenticated read | read group `RequirePerm(pub, "read")` | same middleware | n/a | `cmd/governance/main.go:78`; `cmd/governance/main.go:304-320` |
| `/current-oncall` | GET | authenticated read | read group `RequirePerm(pub, "read")` | same middleware | n/a | `cmd/governance/main.go:78`; `cmd/governance/main.go:321-330` |
| `/forecast/refresh` | POST | authenticated write | workflow group uses `RequireAnyPerm(pub, "replay:write", "*")` | same middleware | global CSRF middleware | `cmd/governance/main.go:80`; `cmd/governance/main.go:331-342` |

## Web Next route handlers

Proxy note:

- The Next proxy layer is not an auth boundary.
- It forwards request headers and cookies to upstream services and relies on the upstream service for auth, RBAC, and CSRF.
- Evidence: `web/app/api/proxy/[service]/[...path]/route.ts:26-39`

| Route | Method | Class | Auth enforcement | RBAC enforcement | CSRF enforcement | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| `/api/proxy/[service]/[...path]` | GET | UNKNOWN | none locally; forwarded upstream | none locally; forwarded upstream | none locally; forwarded upstream | `web/app/api/proxy/[service]/[...path]/route.ts:26-48` |
| `/api/proxy/[service]/[...path]` | POST | UNKNOWN | none locally; forwarded upstream | none locally; forwarded upstream | none locally; forwarded upstream | `web/app/api/proxy/[service]/[...path]/route.ts:26-48` |
| `/api/proxy/[service]/[...path]` | PUT | UNKNOWN | none locally; forwarded upstream | none locally; forwarded upstream | none locally; forwarded upstream | `web/app/api/proxy/[service]/[...path]/route.ts:26-48` |
| `/api/proxy/[service]/[...path]` | PATCH | UNKNOWN | none locally; forwarded upstream | none locally; forwarded upstream | none locally; forwarded upstream | `web/app/api/proxy/[service]/[...path]/route.ts:26-48` |
| `/api/proxy/[service]/[...path]` | DELETE | UNKNOWN | none locally; forwarded upstream | none locally; forwarded upstream | none locally; forwarded upstream | `web/app/api/proxy/[service]/[...path]/route.ts:26-48` |
| `/api/sse/alerts` | GET | public read | none locally; upstream alert SSE requires forwarded cookie for actual data | upstream `alerts` enforces `read` | n/a | `web/app/api/sse/alerts/route.ts:1-8`; `web/app/api/sse/proxy.ts:7-20`; `cmd/alerts/main.go:114` |
| `/api/sse/queue` | GET | public read | none locally; upstream query SSE requires forwarded cookie for actual data | upstream `query` enforces `read` | n/a | `web/app/api/sse/queue/route.ts:1-8`; `web/app/api/sse/proxy.ts:7-20`; `cmd/query/main.go:150-151`; `cmd/query/main.go:612-619` |
| `/api/sse/command-center` | GET | public read | none locally; upstream query SSE requires forwarded cookie for actual data | upstream `query` enforces `read` | n/a | `web/app/api/sse/command-center/route.ts:1-8`; `web/app/api/sse/proxy.ts:7-20`; `cmd/query/main.go:150-151`; `cmd/query/main.go:620-623` |
| `/api/sse/trust` | GET | public read | none locally; upstream query SSE requires forwarded cookie for actual data | upstream `query` enforces `read` | n/a | `web/app/api/sse/trust/route.ts:1-8`; `web/app/api/sse/proxy.ts:7-20`; `cmd/query/main.go:150-151`; `cmd/query/main.go:624-627` |

## Highest-risk route findings

- The only confirmed write route without a CSRF layer in current Go router wiring is `POST /dev/backfill` in `query`.
- `POST /auth/logout` is not explicitly authenticated and can run as a cookie-clearing public write when no valid auth context is present.
- The Next proxy accepts all write verbs locally and is only as safe as the upstream service it forwards to.
