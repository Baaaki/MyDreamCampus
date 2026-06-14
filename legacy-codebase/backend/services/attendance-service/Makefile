.PHONY: sqlc migrate-create migrate-up migrate-down migrate-status build run check-infra dev

# Load environment variables from .env file
include .env.example
export

# Database configuration (loaded from .env)
DB_URL ?= postgres://postgres:postgres@localhost:5437/mydreamcampus_attendance?sslmode=disable

# SQLC
sqlc:
	sqlc generate

# Database migrations
migrate-create:
	goose -dir sql/migrations create $(name) sql

migrate-up:
	goose -dir sql/migrations postgres "$(DB_URL)" up

migrate-down:
	goose -dir sql/migrations postgres "$(DB_URL)" down

migrate-status:
	goose -dir sql/migrations postgres "$(DB_URL)" status

# Build
build:
	go build -o bin/attendance-service cmd/main.go

run:
	air

# Check if infrastructure is running
check-infra:
	@echo "Checking if infrastructure is running..."
	@sudo docker ps | grep mydreamcampus-postgres > /dev/null || (echo "❌ Infrastructure not running! Run 'cd infrastructure && sudo docker compose up -d'" && exit 1)
	@sudo docker ps | grep mydreamcampus-rabbitmq > /dev/null || (echo "❌ RabbitMQ not running! Run 'cd infrastructure && sudo docker compose up -d'" && exit 1)
	@sudo docker ps | grep mydreamcampus-redis > /dev/null || (echo "❌ Redis not running! Run 'cd infrastructure && sudo docker compose up -d'" && exit 1)
	@echo "✅ Infrastructure is running"

# Development workflow
dev: check-infra migrate-up run
