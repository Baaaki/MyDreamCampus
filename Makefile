.PHONY: help dev up down stop infra backend frontend mobile status logs clean \
	test test-backend test-frontend test-mobile test-backend-shared test-backend-services \
	test-coverage

# Default target
help:
	@echo "MyDreamCampus — development commands"
	@echo ""
	@echo "  make up           Bring up infrastructure + backend services (hot-reload)"
	@echo "  make down         Stop backend services + infrastructure"
	@echo ""
	@echo "  make infra        Start only infrastructure (Postgres, RabbitMQ, Redis, Traefik)"
	@echo "  make backend      Start only backend Go services (requires infra)"
	@echo "  make frontend     Install deps and run Vite dev server"
	@echo "  make mobile       Install deps and run Expo dev server"
	@echo ""
	@echo "  make test            Run ALL test suites (backend + frontend + mobile)"
	@echo "  make test-backend    Run Go test ./... across shared + 9 services (with -race)"
	@echo "  make test-frontend   Run Vitest unit tests in frontend/"
	@echo "  make test-mobile     Run Jest unit tests in mobile/"
	@echo "  make test-coverage   Backend tests with coverage report"
	@echo ""
	@echo "  make status       Show running backend services"
	@echo "  make logs         Tail backend service logs"
	@echo "  make clean        Stop everything and prune local volumes"

# Full stack: infra + backend
up: infra backend
	@echo ""
	@echo "Backend + infra running. Next:"
	@echo "  make frontend   — in a new terminal"
	@echo "  make mobile     — in a new terminal"

down:
	@$(MAKE) -C backend stop
	@$(MAKE) -C backend infra-down

dev: up

infra:
	@$(MAKE) -C backend infra

backend:
	@$(MAKE) -C backend dev

frontend:
	@cd frontend && bun install && bun dev

mobile:
	@cd mobile && npm install && npm start

status:
	@$(MAKE) -C backend status

logs:
	@$(MAKE) -C backend logs

stop: down

clean: down
	@echo "Pruning local volumes (docker)..."
	@cd backend/infrastructure && sudo docker compose down -v

# ─────────────────────────────────────────────
# Test targets
# ─────────────────────────────────────────────

test: test-backend test-frontend test-mobile
	@echo ""
	@echo "✓ All test suites passed"

test-backend: test-backend-shared test-backend-services

test-backend-shared:
	@echo "→ shared"
	@cd backend/shared && go test -race -count=1 ./...

test-backend-services:
	@for d in backend/services/*/; do \
		echo "→ $$(basename $$d)"; \
		(cd $$d && go test -race -count=1 ./...) || exit 1; \
	done

test-frontend:
	@echo "→ frontend (vitest)"
	@cd frontend && bun run test

test-mobile:
	@echo "→ mobile (jest)"
	@cd mobile && npm test -- --ci

test-coverage:
	@echo "→ shared (coverage)"
	@cd backend/shared && go test -race -count=1 -coverprofile=coverage.out ./... \
		&& go tool cover -func=coverage.out | tail -1
	@for d in backend/services/*/; do \
		echo "→ $$(basename $$d) (coverage)"; \
		(cd $$d && go test -race -count=1 -coverprofile=coverage.out ./... \
			&& go tool cover -func=coverage.out | tail -1) || exit 1; \
	done
