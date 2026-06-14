.PHONY: help build run test test-cov tidy lint sqcl-all migrate-up-all migrate-down-all migrate-status-all docker-build clean

# Load .env if present so DB_URL etc. are available to migrate targets.
-include .env
export

# Defaults — can be overridden via .env or environment.
DB_URL ?= postgres://postgres:postgres@localhost:5432/mydreamcampus?sslmode=disable
PORT ?= 8080
ENVIRONMENT ?= development

# Modules listed in plan section 3. New modules added here become reachable
# via `make sqlc-<module>`, `make migrate-up-<module>`, etc.
MODULES := auth staff student course_catalog enrollment attendance grades meal payment

# Colors
YELLOW := \033[1;33m
GREEN  := \033[0;32m
RED    := \033[0;31m
NC     := \033[0m

help:
	@echo "$(YELLOW)Monolith — usage:$(NC)"
	@echo "  make build                Build cmd/main.go to bin/monolith"
	@echo "  make run                  Run monolith (with .env)"
	@echo "  make test                 Run all monolith tests"
	@echo "  make test-cov             Run tests with coverage"
	@echo "  make tidy                 go mod tidy"
	@echo "  make lint                 go vet + golangci-lint (if installed)"
	@echo ""
	@echo "$(YELLOW)Per-module sqlc / migrations (substitute <module>):$(NC)"
	@echo "  make sqlc-<module>        Run sqlc generate for one module"
	@echo "  make sqlc-all             Run sqlc for every module that has sqlc.yaml"
	@echo "  make migrate-up-<module>  goose up for one module"
	@echo "  make migrate-up-all       goose up for every module"
	@echo "  make migrate-down-<module>"
	@echo "  make migrate-status-<module>"
	@echo ""
	@echo "$(YELLOW)Modules:$(NC) $(MODULES)"

# ----- Build & run -----

build:
	@echo "$(YELLOW)Building monolith...$(NC)"
	@mkdir -p bin
	go build -o bin/monolith ./cmd/main.go
	@echo "$(GREEN)bin/monolith$(NC)"

run:
	go run ./cmd/main.go

test:
	go test ./...

test-cov:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -5

tidy:
	go mod tidy

lint:
	go vet ./...
	@command -v golangci-lint >/dev/null && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

# ----- Per-module sqlc -----
# Usage: make sqlc-auth, make sqlc-staff, etc.
# Each module owns internal/modules/<module>/sqlc.yaml.

sqlc-%:
	@MODULE=$*; \
	YAML=internal/modules/$$MODULE/sqlc.yaml; \
	if [ ! -f $$YAML ]; then \
		echo "$(RED)$$YAML not found — module not migrated yet$(NC)"; exit 1; \
	fi; \
	echo "$(YELLOW)sqlc generate for $$MODULE$(NC)"; \
	cd internal/modules/$$MODULE && sqlc generate

sqlc-all:
	@for m in $(MODULES); do \
		YAML=internal/modules/$$m/sqlc.yaml; \
		if [ -f $$YAML ]; then \
			echo "$(YELLOW)sqlc generate for $$m$(NC)"; \
			(cd internal/modules/$$m && sqlc generate) || exit 1; \
		else \
			echo "skip $$m (no sqlc.yaml)"; \
		fi; \
	done

# ----- Per-module migrations (goose) -----
# Per plan: tek PostgreSQL DB, modul basina ayri schema. goose tum migration'lari
# ayni DB_URL'e uygular ama her modulun migration klasoru ayri.

migrate-up-%:
	@MODULE=$*; \
	DIR=internal/modules/$$MODULE/sql/migrations; \
	if [ ! -d $$DIR ]; then echo "$(RED)$$DIR not found$(NC)"; exit 1; fi; \
	echo "$(YELLOW)goose up — $$MODULE$(NC)"; \
	goose -dir $$DIR -table goose_db_version_$$MODULE postgres "$(DB_URL)" up

migrate-down-%:
	@MODULE=$*; \
	DIR=internal/modules/$$MODULE/sql/migrations; \
	if [ ! -d $$DIR ]; then echo "$(RED)$$DIR not found$(NC)"; exit 1; fi; \
	echo "$(YELLOW)goose down — $$MODULE$(NC)"; \
	goose -dir $$DIR -table goose_db_version_$$MODULE postgres "$(DB_URL)" down

migrate-status-%:
	@MODULE=$*; \
	DIR=internal/modules/$$MODULE/sql/migrations; \
	if [ ! -d $$DIR ]; then echo "$(RED)$$DIR not found$(NC)"; exit 1; fi; \
	goose -dir $$DIR -table goose_db_version_$$MODULE postgres "$(DB_URL)" status

migrate-up-all:
	@for m in $(MODULES); do \
		DIR=internal/modules/$$m/sql/migrations; \
		if [ -d $$DIR ]; then \
			echo "$(YELLOW)goose up — $$m$(NC)"; \
			goose -dir $$DIR -table goose_db_version_$$m postgres "$(DB_URL)" up || exit 1; \
		fi; \
	done

migrate-down-all:
	@for m in $(MODULES); do \
		DIR=internal/modules/$$m/sql/migrations; \
		if [ -d $$DIR ]; then \
			echo "$(YELLOW)goose down — $$m$(NC)"; \
			goose -dir $$DIR -table goose_db_version_$$m postgres "$(DB_URL)" down || exit 1; \
		fi; \
	done

migrate-create-%:
	@MODULE=$*; \
	if [ -z "$(name)" ]; then echo "Usage: make migrate-create-<module> name=create_users"; exit 1; fi; \
	DIR=internal/modules/$$MODULE/sql/migrations; \
	mkdir -p $$DIR; \
	goose -dir $$DIR create $(name) sql

# ----- Docker -----

docker-build:
	docker build -t mydreamcampus-monolith:latest -f Dockerfile ../

clean:
	rm -rf bin coverage.out
