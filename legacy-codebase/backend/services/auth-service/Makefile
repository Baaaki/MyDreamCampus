.PHONY: sqlc migrate-create migrate-up migrate-down migrate-status build docker-build help

# Load environment variables from .env file
include .env
export

# Database configuration (loaded from .env)
# Default fallback if .env is missing
DB_URL ?= postgres://postgres:postgres@localhost:5432/mydreamcampus_auth?sslmode=disable

# Colors for output
YELLOW := \033[1;33m
NC := \033[0m # No Color

help:
	@echo "$(YELLOW)Available commands:$(NC)"
	@echo "  make sqlc              - Generate Go code from SQL"
	@echo "  make migrate-create    - Create new migration (usage: make migrate-create name=create_users)"
	@echo "  make migrate-up        - Run all pending migrations"
	@echo "  make migrate-down      - Rollback last migration"
	@echo "  make migrate-status    - Show migration status"
	@echo "  make build             - Build the service"
	@echo "  make docker-build      - Build Docker image"

sqlc:
	@echo "$(YELLOW)Generating sqlc code...$(NC)"
	sqlc generate

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required. Usage: make migrate-create name=create_users"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Creating migration: $(name)$(NC)"
	@COUNT=$$(ls -1 sql/migrations/*.sql 2>/dev/null | wc -l); \
	NEXT=$$(printf "%05d" $$((COUNT + 1))); \
	goose -dir sql/migrations create $(name) sql

migrate-up:
	@echo "$(YELLOW)Running migrations...$(NC)"
	goose -dir sql/migrations postgres "$(DB_URL)" up

migrate-down:
	@echo "$(YELLOW)Rolling back last migration...$(NC)"
	goose -dir sql/migrations postgres "$(DB_URL)" down

migrate-status:
	@echo "$(YELLOW)Migration status:$(NC)"
	goose -dir sql/migrations postgres "$(DB_URL)" status

build:
	@echo "$(YELLOW)Building auth-service...$(NC)"
	go build -o bin/auth-service ./cmd/main.go

docker-build:
	@echo "$(YELLOW)Building Docker image...$(NC)"
	docker build -t auth-service:latest .
