# syntax=docker/dockerfile:1
# Build context is the REPO ROOT (Openship's compose builder roots its one
# shared context at the checkout and ignores per-service contexts), so COPY
# sources carry the new-backend/ prefix while destinations keep the old layout
# — the module still reaches ../../shared, its go.mod replace target.

FROM golang:1.26 AS build
ENV GOWORK=off CGO_ENABLED=0
WORKDIR /src

# Module files before sources: a change under shared/ used to invalidate this
# layer, so all ten services re-resolved the same dependency set on every
# shared edit. Only shared's go.mod/go.sum are needed to resolve the graph —
# the replace target's code is copied further down.
COPY new-backend/shared/go.mod new-backend/shared/go.sum ./shared/
COPY new-backend/services/attendance-service/go.mod new-backend/services/attendance-service/go.sum ./services/attendance-service/
WORKDIR /src/services/attendance-service
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY new-backend/shared/ ./shared/
COPY new-backend/services/attendance-service/ ./
# Without these mounts a whole-stack build is thirteen cold compiles of one
# dependency tree — gin, pgx, zap and the rest, once per service, in parallel.
# The caches live outside the image, so they are shared across services and
# across runs; BuildKit's default sharing lets concurrent builds use them.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/attendance ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/attendance /app/attendance
EXPOSE 8086
USER nonroot:nonroot
ENTRYPOINT ["/app/attendance"]
