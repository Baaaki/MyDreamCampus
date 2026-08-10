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
COPY new-backend/services/notification-service/go.mod new-backend/services/notification-service/go.sum ./services/notification-service/
WORKDIR /src/services/notification-service
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Absolute destination: WORKDIR is the service dir by now, so a relative
# ./shared/ would land inside it and leave the replace target (../../shared)
# holding nothing but the go.mod copied above.
COPY new-backend/shared/ /src/shared/
COPY new-backend/services/notification-service/ ./
# Without these mounts a whole-stack build is thirteen cold compiles of one
# dependency tree — gin, pgx, zap and the rest, once per service, in parallel.
# The caches live outside the image, so they are shared across services and
# across runs; BuildKit's default sharing lets concurrent builds use them.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/notification ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/notification /app/notification
# Email templates are loaded with a CWD-relative glob (internal/templates/*.html),
# so they must sit next to the binary under /app.
COPY --from=build /src/services/notification-service/internal/templates /app/internal/templates
EXPOSE 9090
USER nonroot:nonroot
ENTRYPOINT ["/app/notification"]
