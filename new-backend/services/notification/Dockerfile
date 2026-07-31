# syntax=docker/dockerfile:1
# Build context is the REPO ROOT (Openship's compose builder roots its one
# shared context at the checkout and ignores per-service contexts), so COPY
# sources carry the new-backend/ prefix while destinations keep the old layout
# — the module still reaches ../../shared, its go.mod replace target.

FROM golang:1.26 AS build
ENV GOWORK=off CGO_ENABLED=0
WORKDIR /src

COPY new-backend/shared/ ./shared/
COPY new-backend/services/notification/go.mod new-backend/services/notification/go.sum ./services/notification/
WORKDIR /src/services/notification
RUN go mod download

COPY new-backend/services/notification/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/notification ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/notification /app/notification
# Email templates are loaded with a CWD-relative glob (internal/templates/*.html),
# so they must sit next to the binary under /app.
COPY --from=build /src/services/notification/internal/templates /app/internal/templates
EXPOSE 9090
USER nonroot:nonroot
ENTRYPOINT ["/app/notification"]
