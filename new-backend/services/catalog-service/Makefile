.PHONY: sqlc migrate-up migrate-down migrate-status build run test

# Database is `catalog`, schema and goose table stay `course_catalog` — the
# migration history applied in phase 1 keys on the module name.
DB_URL ?= postgres://catalog_svc:$(SERVICE_DB_PASSWORD)@localhost:5432/catalog?sslmode=disable
GOOSE_TABLE := goose_db_version_course_catalog

sqlc:
	sqlc generate

migrate-up:
	goose -dir internal/sql/migrations -table $(GOOSE_TABLE) postgres "$(DB_URL)" up

migrate-down:
	goose -dir internal/sql/migrations -table $(GOOSE_TABLE) postgres "$(DB_URL)" down

migrate-status:
	goose -dir internal/sql/migrations -table $(GOOSE_TABLE) postgres "$(DB_URL)" status

build:
	go build -o bin/catalog ./cmd

run:
	go run ./cmd

test:
	go test ./...
