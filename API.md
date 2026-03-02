# API Overview

## Gateway
- `POST /auth/login`
- `POST /auth/logout`
- `GET /me`
- `GET /audit/verify` (admin)

## Query
- `GET /alerts`
- `GET /scores`

## Alerts
- `GET /sse/alerts`
- `POST /alerts/{id}/ack`

## Governance
- `POST /models/deploy`
- `POST /replay/start`
- `GET /models`
