# syntax=docker/dockerfile:1
# Build context is the REPO ROOT: this module's go.mod replaces the shared
# module with ../../shared, which sits outside the service directory and so is
# unreachable from a per-service context. COPY sources therefore carry the
# new-backend/ prefix while destinations keep the old layout.

FROM golang:1.26 AS build
ENV GOWORK=off CGO_ENABLED=0
WORKDIR /src

# Module files before sources: a change under shared/ used to invalidate this
# layer, so all ten services re-resolved the same dependency set on every
# shared edit. Only shared's go.mod/go.sum are needed to resolve the graph —
# the replace target's code is copied further down.
COPY new-backend/shared/go.mod new-backend/shared/go.sum ./shared/
COPY new-backend/services/payment-service/go.mod new-backend/services/payment-service/go.sum ./services/payment-service/
WORKDIR /src/services/payment-service
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Absolute destination: WORKDIR is the service dir by now, so a relative
# ./shared/ would land inside it and leave the replace target (../../shared)
# holding nothing but the go.mod copied above.
COPY new-backend/shared/ /src/shared/
COPY new-backend/services/payment-service/ ./
# Without these mounts a whole-stack build is thirteen cold compiles of one
# dependency tree — gin, pgx, zap and the rest, once per service, in parallel.
# The caches live outside the image, so they are shared across services and
# across runs; BuildKit's default sharing lets concurrent builds use them.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/payment ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/payment /app/payment
EXPOSE 8089
USER nonroot:nonroot
ENTRYPOINT ["/app/payment"]
