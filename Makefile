COMPOSE=docker compose -f deploy/docker/docker-compose.yml
FAST_COMPOSE=docker compose -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.fast.yml

.PHONY: help doctor lint dev-keys up down migrate topics seed smoke demo demo-fast screenshot-smoke browser-validate verify-screenshots test unit train-models backfill integration-test integration-suite replay-test security-test ml-proof demo-smoke verify release-gate prod-readiness perf-smoke boot-repeat logs status

help: ## Show available targets
	@echo "Sentinel Platform - Available targets:"
	@echo "  make unit             Run Go + Python + Web tests from repo root"
	@echo "  make verify           Run unit + security + replay + demo smoke"

doctor:
	./scripts/doctor.sh

lint:
	./scripts/lint.sh

dev-keys:
	./scripts/dev-keys.sh

up:
	$(COMPOSE) up -d --build

up-fast:
	$(FAST_COMPOSE) up -d --build

down:
	$(COMPOSE) down -v

migrate:
	./scripts/migrate.sh

topics:
	./scripts/create-topics.sh

seed:
	POSTGRES_URL=postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable go run ./scripts/seed-users.go

smoke:
	./scripts/smoke.sh

demo:
	./scripts/run-demo.sh default

demo-fast:
	./scripts/run-demo.sh fast

screenshot-smoke:
	./scripts/capture-screenshots.sh

browser-validate:
	./scripts/browser-validate.sh

verify-screenshots:
	node scripts/verify-screenshot-manifest.mjs docs/screenshots/manifest.json

test:
	go test -race -count=1 ./...

unit:
	go test ./...
	PYTHONPATH=services/inference python3 -m pytest -q services/inference/tests services/inference/test_app.py
	cd web && npm test -- --runInBand

train-models:
	python3 services/inference/train/train_models.py

backfill:
	POSTGRES_URL=postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable YEARS=$(or $(YEARS),20) go run ./scripts/backfill.go

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

logs:
	$(COMPOSE) logs -f --tail=50

status:
	$(COMPOSE) ps
