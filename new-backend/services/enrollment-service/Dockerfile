# syntax=docker/dockerfile:1
# Build context is the REPO ROOT (Openship's compose builder roots its one
# shared context at the checkout and ignores per-service contexts), so COPY
# sources carry the new-backend/ prefix while destinations keep the old layout
# — the module still reaches ../../shared, its go.mod replace target.

FROM golang:1.26 AS build
ENV GOWORK=off CGO_ENABLED=0
WORKDIR /src

COPY new-backend/shared/ ./shared/
COPY new-backend/services/enrollment-service/go.mod new-backend/services/enrollment-service/go.sum ./services/enrollment-service/
WORKDIR /src/services/enrollment-service
RUN go mod download

COPY new-backend/services/enrollment-service/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/enrollment ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/enrollment /app/enrollment
EXPOSE 8085
USER nonroot:nonroot
ENTRYPOINT ["/app/enrollment"]
