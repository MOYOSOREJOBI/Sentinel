COMPOSE        = docker compose -f deploy/docker/docker-compose.yml
DEV_COMPOSE    = docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.dev.yml
FAST_COMPOSE   = docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.fast.yml
LOCAL_COMPOSE  = docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.override.local.yml
LOCAL_DEV_COMPOSE = docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.dev.yml -f deploy/docker/docker-compose.override.local.yml
LOCAL_FAST_COMPOSE = docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.fast.yml -f deploy/docker/docker-compose.override.local.yml

POSTGRES_PORT ?= 5432
POSTGRES_USER ?= sentinel
POSTGRES_DB   ?= sentinel
POSTGRES_URL  ?= postgres://$(POSTGRES_USER):sentinel@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

.PHONY: help doctor lint dev-keys \
        up up-dev up-local up-local-dev up-fast up-fast-local \
        down down-local down-preserve \
        migrate topics seed smoke \
        demo demo-local demo-fast demo-fast-local \
        screenshot-smoke browser-validate verify-screenshots \
        test unit train-models backfill \
        integration-test integration-suite replay-test security-test ml-proof demo-smoke \
        verify release-gate prod-readiness perf-smoke boot-repeat \
        backup restore \
        logs logs-local status status-local

help: ## Show available targets
	@echo "Sentinel Platform — Available targets:"
	@echo ""
	@echo "  Development (hot-reload):"
	@echo "    make dev             Boot stack with Next.js dev server (hot-reload)"
	@echo "    make dev-local       Same, using alternate ports to avoid conflicts"
	@echo ""
	@echo "  Production:"
	@echo "    make up              Boot stack with production Next.js build + nginx TLS"
	@echo "    make up-local        Same, using alternate ports"
	@echo "    make demo            Full demo: keys + up + seed + smoke"
	@echo ""
	@echo "  Data:"
	@echo "    make backup          Dump postgres to backups/sentinel-<ts>.sql.gz"
	@echo "    make restore FILE=<> Restore postgres from a backup file"
	@echo "    make down            Tear down (data preserved in named volumes)"
	@echo "    make down-wipe       Tear down and DELETE all persistent data"
	@echo ""
	@echo "  Testing:"
	@echo "    make unit            Go + Python + Web unit tests"
	@echo "    make verify          unit + security + replay + demo-smoke"
	@echo "    make release-gate    Full gate: unit + replay + security + browser"

doctor:
	./scripts/doctor.sh

lint:
	./scripts/lint.sh

dev-keys:
	./scripts/dev-keys.sh

# ─── Production stack (Dockerfile.web + nginx TLS) ────────────────────────────
up:
	$(MAKE) dev-keys
	$(COMPOSE) up -d --build

up-local:
	$(MAKE) dev-keys
	$(LOCAL_COMPOSE) up -d --build

# ─── Development stack (hot-reload web, no nginx TLS requirement) ─────────────
dev:
	$(MAKE) dev-keys
	$(DEV_COMPOSE) up -d --build

dev-local:
	$(MAKE) dev-keys
	$(LOCAL_DEV_COMPOSE) up -d --build

# Keep old aliases working
up-fast:
	$(MAKE) dev-keys
	$(FAST_COMPOSE) up -d --build

up-fast-local:
	$(MAKE) dev-keys
	$(LOCAL_FAST_COMPOSE) up -d --build

# ─── Teardown ─────────────────────────────────────────────────────────────────
down:
	$(COMPOSE) down

down-local:
	$(LOCAL_COMPOSE) down

down-wipe:
	$(COMPOSE) down -v

# ─── Database / messaging ─────────────────────────────────────────────────────
migrate:
	./scripts/migrate.sh

topics:
	./scripts/create-topics.sh

seed:
	POSTGRES_URL=$(POSTGRES_URL) go run -tags seedusers ./scripts/seed-users.go

backup:
	./scripts/backup-db.sh backup

restore:
	./scripts/backup-db.sh restore $(FILE)

# ─── Smoke / demo ────────────────────────────────────────────────────────────
smoke:
	./scripts/smoke.sh

demo:
	./scripts/run-demo.sh default

demo-local:
	SENTINEL_USE_LOCAL_OVERRIDE=1 ./scripts/run-demo.sh default

demo-fast:
	./scripts/run-demo.sh fast

demo-fast-local:
	SENTINEL_USE_LOCAL_OVERRIDE=1 ./scripts/run-demo.sh fast

screenshot-smoke:
	./scripts/capture-screenshots.sh

browser-validate:
	./scripts/browser-validate.sh

verify-screenshots:
	node scripts/verify-screenshot-manifest.mjs docs/screenshots/manifest.json

# ─── Tests ───────────────────────────────────────────────────────────────────
test:
	go test -race -count=1 ./...

unit:
	go test ./...
	PYTHONPATH=services/inference python3 -m pytest -q services/inference/tests services/inference/test_app.py
	cd web && npm test -- --runInBand

train-models:
	python3 services/inference/train/train_models.py

backfill:
	POSTGRES_URL=$(POSTGRES_URL) YEARS=$(or $(YEARS),20) go run -tags backfill ./scripts/backfill.go

integration-test:
	./scripts/integration-test.sh

integration-suite:
	./integration/e2e_pipeline_test.sh
	./integration/idempotency_test.sh
	./integration/ml_scores_persist_test.sh
	./integration/replay_equivalence_test.sh
	./integration/startup_ordering_test.sh
	./integration/dependency_failure_test.sh
	./integration/case_workflow_test.sh

replay-test:
	./integration/replay_equivalence_test.sh

security-test:
	./integration/dependency_failure_test.sh
	./integration/case_workflow_test.sh

ml-proof:
	./integration/ml_scores_persist_test.sh

demo-smoke:
	./integration/e2e_pipeline_test.sh
	./integration/ml_scores_persist_test.sh
	./integration/case_workflow_test.sh

verify:
	$(MAKE) unit
	$(MAKE) security-test
	$(MAKE) replay-test
	$(MAKE) demo-smoke

release-gate:
	$(MAKE) unit
	$(MAKE) replay-test
	$(MAKE) security-test
	$(MAKE) demo-smoke
	REQUIRE_BROWSER=1 $(MAKE) browser-validate
	REQUIRE_BROWSER=1 $(MAKE) screenshot-smoke
	$(MAKE) verify-screenshots

prod-readiness:
	./scripts/prod-readiness.sh

perf-smoke:
	./scripts/perf-smoke.sh

boot-repeat:
	./scripts/boot-repeat.sh 5

# ─── Logs / status ───────────────────────────────────────────────────────────
logs:
	$(COMPOSE) logs -f --tail=50

logs-local:
	$(LOCAL_COMPOSE) logs -f --tail=50

status:
	$(COMPOSE) ps

status-local:
	$(LOCAL_COMPOSE) ps
