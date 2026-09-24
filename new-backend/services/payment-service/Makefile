.PHONY: sqlc migrate-up migrate-down migrate-status build run test

DB_URL ?= postgres://payment_svc:$(SERVICE_DB_PASSWORD)@localhost:5432/payment?sslmode=disable
GOOSE_TABLE := goose_db_version_payment

sqlc:
	sqlc generate

migrate-up:
	goose -dir internal/sql/migrations -table $(GOOSE_TABLE) postgres "$(DB_URL)" up

migrate-down:
	goose -dir internal/sql/migrations -table $(GOOSE_TABLE) postgres "$(DB_URL)" down

migrate-status:
	goose -dir internal/sql/migrations -table $(GOOSE_TABLE) postgres "$(DB_URL)" status

build:
	go build -o bin/payment ./cmd

run:
	go run ./cmd

test:
	go test ./...
