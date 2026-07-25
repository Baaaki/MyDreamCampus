.PHONY: help dev up down stop infra infra-down backend notification frontend mobile clean \
	test test-backend test-frontend test-mobile test-coverage \
	deploy deploy-down deploy-logs deploy-ps deploy-update check-env

INFRA   := new-backend/infrastructure
COMPOSE := $(INFRA)/docker-compose.yml

# Empty when the docker socket is already reachable (root, or user in the
# `docker` group) so the server never prompts for a password; falls back to
# sudo on machines where it isn't. Evaluated once per make run.
SUDO := $(shell docker info >/dev/null 2>&1 || echo sudo)

# Default target
help:
	@echo "MyDreamCampus — commands"
	@echo ""
	@echo "SERVER (tek komut — frontend + backend + infra, hepsi container'da):"
	@echo "  make deploy        Build + start the whole stack (SPA served by caddy on :80)"
	@echo "  make deploy-update git pull + rebuild changed services + restart"
	@echo "  make deploy-logs   Follow monolith + caddy logs"
	@echo "  make deploy-ps     Show container status"
	@echo "  make deploy-down   Stop everything (volumes/data kept)"
	@echo ""
	@echo "LOCAL DEV (hot reload, ayri terminaller):"
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
	@echo "  Docker erisimi: $(if $(SUDO),sudo ile (sifre sorar) — 'sudo usermod -aG docker $$USER' + yeniden giris ile kalicilastir,dogrudan (sudo gerekmiyor))"

# Full stack: infra + backend
up: infra backend

down: infra-down

dev: up

infra:
	$(SUDO) docker compose -f $(COMPOSE) up -d

infra-down:
	$(SUDO) docker compose -f $(COMPOSE) down

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
	$(SUDO) docker compose -f $(COMPOSE) down -v

# ─────────────────────────────────────────────
# Deploy targets — one command for the full stack.
# The SPA is built into the caddy image, so there is no separate frontend
# process to start: caddy serves /srv and proxies /api to the monolith.
# ─────────────────────────────────────────────

# Fail early with a readable message instead of compose's raw variable errors.
check-env:
	@test -f $(INFRA)/.env || { \
		echo "HATA: $(INFRA)/.env yok."; \
		echo "  cp $(INFRA)/.env.example $(INFRA)/.env && nano $(INFRA)/.env"; \
		exit 1; }
	@grep -q 'CHANGE_ME' $(INFRA)/.env && { \
		echo "HATA: $(INFRA)/.env icinde hala CHANGE_ME var — secret'lari doldur."; \
		echo "  openssl rand -base64 48"; \
		exit 1; } || true

deploy: check-env
	$(SUDO) docker compose -f $(COMPOSE) up -d --build

deploy-update: check-env
	git pull
	$(SUDO) docker compose -f $(COMPOSE) up -d --build

deploy-logs:
	$(SUDO) docker compose -f $(COMPOSE) logs -f monolith caddy

deploy-ps:
	$(SUDO) docker compose -f $(COMPOSE) ps

deploy-down:
	$(SUDO) docker compose -f $(COMPOSE) down

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
