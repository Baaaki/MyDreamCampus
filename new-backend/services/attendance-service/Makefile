.PHONY: sqlc migrate-up migrate-down migrate-status build run test

# The goose version table keeps the module name, not the database name, so the
# migration history applied in phase 1 stays valid.
DB_URL ?= postgres://attendance_svc:$(SERVICE_DB_PASSWORD)@localhost:5432/attendance?sslmode=disable
GOOSE_TABLE := goose_db_version_attendance

sqlc:
	sqlc generate

migrate-up:
	goose -dir internal/sql/migrations -table $(GOOSE_TABLE) postgres "$(DB_URL)" up

migrate-down:
	goose -dir internal/sql/migrations -table $(GOOSE_TABLE) postgres "$(DB_URL)" down

migrate-status:
	goose -dir internal/sql/migrations -table $(GOOSE_TABLE) postgres "$(DB_URL)" status

build:
	go build -o bin/attendance ./cmd

run:
	go run ./cmd

test:
	go test ./...
