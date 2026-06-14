.PHONY: proto build run clean tidy

# Generate protobuf files
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/payment.proto

# Build the service
build: proto
	go build -o bin/payment-service ./cmd/main.go

# Run the service
run:
	go run ./cmd/main.go

# Clean build artifacts
clean:
	rm -rf bin/

# Tidy dependencies
tidy:
	go mod tidy
