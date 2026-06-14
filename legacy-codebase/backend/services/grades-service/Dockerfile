# Build context: backend/
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy dependency files first for layer caching
COPY shared/go.mod shared/go.sum shared/
COPY services/grades-service/go.mod services/grades-service/go.sum services/grades-service/
RUN cd services/grades-service && go mod download

# Copy source code
COPY shared/ shared/
COPY services/grades-service/ services/grades-service/

# Build static binary
WORKDIR /build/services/grades-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/main.go

# Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /app /app

USER app
EXPOSE 8007
ENTRYPOINT ["/app"]
