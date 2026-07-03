.PHONY: help dev up down stop infra infra-down backend notification frontend mobile clean \
	test test-backend test-frontend test-mobile test-coverage

COMPOSE := new-backend/infrastructure/docker-compose.yml

# Default target
help:
	@echo "MyDreamCampus — development commands"
	@echo ""
	@echo "  make up           Bring up infrastructure + monolith backend"
	@echo "  make down         Stop infrastructure (monolith runs in foreground)"
	@echo ""
	@echo "  make infra        Start only infrastructure (Postgres x2, RabbitMQ, Redis, MailHog)"
	@echo "  make backend      Run the monolith (requires infra)"
	@echo "  make notification Run the notification service (requires infra)"
	@echo "  make frontend     Install deps and run Vite dev server"
	@echo "  make mobile       Install deps and run Expo dev server"
	@echo ""
	@echo "  make test            Run ALL test suites (backend + frontend + mobile)"
	@echo "  make test-backend    Run Go tests across monolith + shared + notification (with -race)"
	@echo "  make test-frontend   Run Vitest unit tests in frontend/"
	@echo "  make test-mobile     Run Jest unit tests in mobile/"
	@echo "  make test-coverage   Backend tests with coverage report"
	@echo ""
	@echo "  make clean        Stop everything and prune local volumes"
	@echo ""
	@echo "  NOT: docker komutlari sudo ister (kullanici docker grubunda degil)."

# Full stack: infra + backend
up: infra backend

down: infra-down

dev: up

infra:
	sudo docker compose -f $(COMPOSE) up -d

infra-down:
	sudo docker compose -f $(COMPOSE) down

backend:
	cd new-backend/monolith && go run ./cmd

notification:
	cd new-backend/services/notification && go run ./cmd

frontend:
	@cd frontend && bun install && bun dev

mobile:
	@cd mobile && npm install && npm start

stop: down

clean: infra-down
	@echo "Pruning local volumes (docker)..."
	sudo docker compose -f $(COMPOSE) down -v

# ─────────────────────────────────────────────
# Test targets
# ─────────────────────────────────────────────

test: test-backend test-frontend test-mobile
	@echo ""
	@echo "✓ All test suites passed"

test-backend:
	@echo "→ monolith + shared + notification"
	@cd new-backend && go test -race -count=1 ./monolith/... ./shared/... ./services/notification/...

test-frontend:
	@echo "→ frontend (vitest)"
	@cd frontend && bun run test

test-mobile:
	@echo "→ mobile (jest)"
	@cd mobile && npm test -- --ci

test-coverage:
	@echo "→ backend (coverage)"
	@cd new-backend && go test -race -count=1 -coverprofile=coverage.out ./monolith/... ./shared/... \
		&& go tool cover -func=coverage.out | tail -1
