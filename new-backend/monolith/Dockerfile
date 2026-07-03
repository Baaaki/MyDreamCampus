# syntax=docker/dockerfile:1
# Build context is new-backend/ so the monolith module can reach ../shared
# (its go.mod replace target). GOWORK=off keeps the build off go.work, which
# also lists services/notification that we do not copy here.

FROM golang:1.26 AS build
ENV GOWORK=off CGO_ENABLED=0
WORKDIR /src

# shared is a local replace target — copy it before resolving deps.
COPY shared/ ./shared/
COPY monolith/go.mod monolith/go.sum ./monolith/
WORKDIR /src/monolith
RUN go mod download

COPY monolith/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/monolith ./cmd/main.go

# Static distroless: pgx and argon2 are pure Go, so CGO is not needed.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/monolith /app/monolith
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/monolith"]
