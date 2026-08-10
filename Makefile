.PHONY: help dev up down stop infra infra-down frontend mobile clean \
	test test-backend test-frontend test-mobile test-coverage \
	deploy deploy-down deploy-logs deploy-ps deploy-update check-env \
	backup restore \
	autodeploy-install autodeploy-status autodeploy-logs autodeploy-now autodeploy-off

INFRA        := new-backend/infrastructure
COMPOSE_FILE := $(INFRA)/docker-compose.yml

# The base file publishes only caddy's :80 so a PaaS (Openship) can front it
# without fighting for :443; the standalone overlay adds the infra loopback
# ports and caddy's :443 back. Every target here runs the stack WITHOUT such a
# platform, so both files are always loaded.
#
# A Prometheus/Grafana/Loki stack is out of scope for now, but it slots in as a
# third overlay without rethinking this variable:
#   $(if $(OBSERVABILITY),-f $(INFRA)/docker-compose.observability.yml)
# turns `make deploy OBSERVABILITY=1` on once that file exists.
COMPOSE := -f $(COMPOSE_FILE) -f $(INFRA)/docker-compose.standalone.yml

# The four containers a locally-run service needs, plus the one-shot migrator.
INFRA_SERVICES := postgres redis rabbitmq mailhog migrate

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
	@echo "  make deploy-logs   Follow caddy + auth + catalog logs"
	@echo "  make deploy-ps     Show container status"
	@echo "  make deploy-down   Stop everything (volumes/data kept)"
	@echo ""
	@echo "TEK SERVIS (digerlerine dokunmadan):"
	@echo "  make deploy-grades   Rebuild + restart one service (any compose service name)"
	@echo "  make logs-grades     Follow one service's log"
	@echo "  make restart-grades  Restart one service"
	@echo ""
	@echo "YEDEK:"
	@echo "  make backup            pg_dumpall of all databases -> $(DB_BACKUP_DIR)"
	@echo "  make restore FILE=...  YIKICI — overwrites the databases from a dump"
	@echo ""
	@echo "OTOMATIK DEPLOY (sunucu origin/main'i yoklar, degisince kendini gunceller):"
	@echo "  make autodeploy-install  Enable the systemd user timer (2 min poll)"
	@echo "  make autodeploy-status   When the next poll runs / last result"
	@echo "  make autodeploy-logs     Follow deploy history"
	@echo "  make autodeploy-now      Poll immediately, don't wait for the timer"
	@echo "  make autodeploy-off      Disable it"
	@echo ""
	@echo "LOCAL DEV (hot reload, ayri terminaller):"
	@echo "  make up           Start infrastructure, then run a service by hand"
	@echo "  make down         Stop everything"
	@echo ""
	@echo "  make infra        Start only infrastructure (Postgres, RabbitMQ, Redis, MailHog, migrate)"
	@echo "  make run-grades   Run one service on the host (requires infra)"
	@echo "  make frontend     Install deps and run Vite dev server"
	@echo "  make mobile       Install deps and run Expo dev server"
	@echo ""
	@echo "  make test            Run ALL test suites (backend + frontend + mobile)"
	@echo "  make test-backend    Run Go tests across shared + every service (with -race)"
	@echo "  make test-frontend   Run Vitest unit tests in frontend/"
	@echo "  make test-mobile     Run Jest unit tests in mobile/"
	@echo "  make test-coverage   Backend tests with coverage report"
	@echo ""
	@echo "  make clean        Stop everything and prune local volumes"
	@echo ""
	@echo "  Docker erisimi: $(if $(SUDO),sudo ile (sifre sorar) — 'sudo usermod -aG docker $$USER' + yeniden giris ile kalicilastir,dogrudan (sudo gerekmiyor))"

up: infra
	@echo ""
	@echo "Infra hazir. Servisi elle calistir:  make run-grades"
	@echo "Hepsini container'da isteyen:        make deploy"

down: infra-down

dev: up

# Only the infra containers: running ten services on the host is not a
# workflow, `make deploy` is. This exists so a single service can be run with
# `go run` against real Postgres/RabbitMQ/Redis.
infra:
	$(SUDO) docker compose $(COMPOSE) up -d $(INFRA_SERVICES)

infra-down:
	$(SUDO) docker compose $(COMPOSE) down

# One service on the host, e.g. `make run-grades` or `make run-notification`.
run-%:
	cd new-backend/services/$*-service && go run ./cmd

frontend:
	@cd frontend && bun install && bun dev

mobile:
	@cd mobile && npm install && npm start

stop: down

clean: infra-down
	@echo "Pruning local volumes (docker)..."
	$(SUDO) docker compose $(COMPOSE) down -v

# ─────────────────────────────────────────────
# Deploy targets — one command for the full stack.
# The SPA is built into the caddy image, so there is no separate frontend
# process to start: caddy serves /srv and proxies each /api prefix to its
# own service.
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

# --remove-orphans: a container whose service left the compose file keeps
# running and keeps holding its host ports, so the next `up` fails on a port
# that looks free. Only on the two whole-stack targets — a single-service
# target must never sweep.
deploy: check-env
	$(SUDO) docker compose $(COMPOSE) up -d --build --remove-orphans

deploy-update: check-env
	git pull
	$(SUDO) docker compose $(COMPOSE) up -d --build --remove-orphans

deploy-logs:
	$(SUDO) docker compose $(COMPOSE) logs -f caddy auth-service catalog-service

deploy-ps:
	$(SUDO) docker compose $(COMPOSE) ps

deploy-down:
	$(SUDO) docker compose $(COMPOSE) down

# ─────────────────────────────────────────────
# Single-service targets — the operational point of the split. Without them
# everyone reaches for `make deploy` and restarts all 16 containers to ship a
# one-line change in one service.
# ─────────────────────────────────────────────

# Compose names the ten Go services `<name>-service`, but the container is
# `mydreamcampus-meal` and `make run-meal` already takes the short name, so
# `make deploy-meal` is what everyone types — and it failed with a bare
# "no such service: meal". Resolve the short form here; anything else (caddy,
# postgres, rabbitmq) passes through untouched.
GO_SERVICES := auth staff student catalog enrollment attendance grades meal payment notification
compose_service = $(if $(filter $1,$(GO_SERVICES)),$1-service,$1)

# Rebuild and restart one service, leaving the other 15 running.
deploy-%: check-env
	$(SUDO) docker compose $(COMPOSE) up -d --no-deps --build $(call compose_service,$*)

logs-%:
	$(SUDO) docker compose $(COMPOSE) logs -f $(call compose_service,$*)

restart-%:
	$(SUDO) docker compose $(COMPOSE) restart $(call compose_service,$*)

# ─────────────────────────────────────────────
# Backup — nine databases, one Postgres container, so one dump covers them all.
# ─────────────────────────────────────────────

DB_BACKUP_DIR ?= $(HOME)/mydreamcampus-backups

backup:
	@mkdir -p $(DB_BACKUP_DIR)
	$(SUDO) docker exec mydreamcampus-postgres pg_dumpall -U postgres \
		| gzip > $(DB_BACKUP_DIR)/all-$$(date +%F-%H%M).sql.gz
	@ls -lh $(DB_BACKUP_DIR) | tail -5

# DESTRUCTIVE — overwrites the current databases with the dump's contents.
restore:
	@test -n "$(FILE)" || { echo "kullanim: make restore FILE=/yol/yedek.sql.gz"; exit 1; }
	gunzip -c $(FILE) | $(SUDO) docker exec -i mydreamcampus-postgres psql -U postgres

# ─────────────────────────────────────────────
# Auto-deploy — a systemd user timer polls origin and redeploys on new commits.
# Pull-based because a home server behind NAT cannot receive a webhook.
# ─────────────────────────────────────────────

UNIT_DIR := $(HOME)/.config/systemd/user

autodeploy-install:
	@mkdir -p $(UNIT_DIR)
	@sed -e 's|@REPO@|$(CURDIR)|g' -e 's|@UID@|$(shell id -u)|g' \
		scripts/systemd/mydreamcampus-deploy.service > $(UNIT_DIR)/mydreamcampus-deploy.service
	@cp scripts/systemd/mydreamcampus-deploy.timer $(UNIT_DIR)/mydreamcampus-deploy.timer
	@chmod +x scripts/auto-deploy.sh
	systemctl --user daemon-reload
	systemctl --user enable --now mydreamcampus-deploy.timer
	@echo ""
	@echo "Otomatik deploy aktif — origin/main her 2 dakikada bir kontrol edilir."
	@echo "  make autodeploy-status   sonraki kontrol ne zaman"
	@echo "  make autodeploy-logs     deploy gecmisi"
	@echo ""
	@echo "NOT: 'sudo loginctl enable-linger $(shell id -un)' yapilmadiysa timer sadece"
	@echo "     sen SSH ile bagliyken calisir."

autodeploy-status:
	@systemctl --user list-timers mydreamcampus-deploy.timer --all
	@systemctl --user status mydreamcampus-deploy.service --no-pager || true

autodeploy-logs:
	journalctl --user -u mydreamcampus-deploy.service -f

# Run one poll right now instead of waiting for the timer.
autodeploy-now:
	systemctl --user start mydreamcampus-deploy.service
	@journalctl --user -u mydreamcampus-deploy.service -n 30 --no-pager

autodeploy-off:
	systemctl --user disable --now mydreamcampus-deploy.timer
	@echo "Otomatik deploy kapatildi. Elle: make deploy-update"

# ─────────────────────────────────────────────
# Test targets
# ─────────────────────────────────────────────

test: test-backend test-frontend test-mobile
	@echo ""
	@echo "✓ All test suites passed"

test-backend:
	@echo "→ shared + every service"
	@cd new-backend/shared && go test -race -count=1 ./...
	@for svc in new-backend/services/*/; do \
		echo "→ $$svc"; \
		(cd $$svc && go test -race -count=1 ./...) || exit 1; \
	done

test-frontend:
	@echo "→ frontend (vitest)"
	@cd frontend && bun run test

test-mobile:
	@echo "→ mobile (jest)"
	@cd mobile && npm test -- --ci

test-coverage:
	@echo "→ backend (coverage, shared only — per-service profiles cannot be merged by go tool cover)"
	@cd new-backend/shared && go test -race -count=1 -coverprofile=coverage.out ./... \
		&& go tool cover -func=coverage.out | tail -1
