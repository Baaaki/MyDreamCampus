.PHONY: sqlc migrate-create migrate-up migrate-down migrate-status build run dev test

# Load environment variables from .env file
-include .env
export

# Database configuration
DB_URL ?= postgres://postgres:postgres@localhost:5438/mydreamcampus_grades?sslmode=disable

# sqlc
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
	go build -o bin/grades-service cmd/main.go

run:
	air

# Development workflow
dev: migrate-up run

# Quick test
test:
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8007/health | jq .
