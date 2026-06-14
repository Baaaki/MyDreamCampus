.PHONY: sqlc migrate-create migrate-up migrate-down migrate-status build run check-infra dev test delete-semesters

# Load .env if exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Database configuration (fallback if .env doesn't exist)
DATABASE_URL ?= postgres://postgres:postgres@localhost:5435/mydreamcampus_catalog?sslmode=disable

# SQLC
sqlc:
	sqlc generate

# Database migrations
migrate-create:
	goose -dir sql/migrations create $(name) sql

migrate-up:
	goose -dir sql/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir sql/migrations postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir sql/migrations postgres "$(DATABASE_URL)" status

# Build
build:
	go build -o bin/course-catalog-service cmd/main.go

run:
	air

# Check if infrastructure is running
check-infra:
	@echo "Checking if infrastructure is running..."
	@sudo docker ps | grep mydreamcampus-postgres > /dev/null || (echo "❌ Infrastructure not running! Run 'cd infrastructure && sudo docker compose up -d'" && exit 1)
	@sudo docker ps | grep mydreamcampus-rabbitmq > /dev/null || (echo "❌ RabbitMQ not running! Run 'cd infrastructure && sudo docker compose up -d'" && exit 1)
	@echo "✅ Infrastructure is running"

# Development workflow
dev: check-infra migrate-up run

# Delete all semesters (test cleanup)
delete-semesters:
	@echo "Deleting all semesters from catalog database..."
	@sudo docker exec mydreamcampus-postgres-catalog psql -U postgres -d mydreamcampus_catalog -c "\
		DELETE FROM course_schedule_sessions; \
		DELETE FROM semester_courses; \
		DELETE FROM semesters;"
	@echo "Done."

# Quick test
test:
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8003/health | jq .
