# syntax=docker/dockerfile:1
# Build context is new-backend/ so the notification module can reach
# ../../shared (its go.mod replace target).

FROM golang:1.26 AS build
ENV GOWORK=off CGO_ENABLED=0
WORKDIR /src

COPY shared/ ./shared/
COPY services/notification/go.mod services/notification/go.sum ./services/notification/
WORKDIR /src/services/notification
RUN go mod download

COPY services/notification/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/notification ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/notification /app/notification
EXPOSE 9090
USER nonroot:nonroot
ENTRYPOINT ["/app/notification"]
