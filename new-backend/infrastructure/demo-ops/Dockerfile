# syntax=docker/dockerfile:1
FROM golang:1.27 AS build
ENV CGO_ENABLED=0
ARG GOOSE_VERSION=v3.28.0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}

FROM postgres:18-alpine
RUN apk add --no-cache redis curl jq tzdata

COPY --from=build /go/bin/goose /usr/local/bin/goose

COPY new-backend/services/auth-service/internal/sql/migrations       /migrations/modules/auth
COPY new-backend/services/staff-service/internal/sql/migrations      /migrations/modules/staff
COPY new-backend/services/student-service/internal/sql/migrations    /migrations/modules/student
COPY new-backend/services/catalog-service/internal/sql/migrations    /migrations/modules/course_catalog
COPY new-backend/services/enrollment-service/internal/sql/migrations /migrations/modules/enrollment
COPY new-backend/services/attendance-service/internal/sql/migrations /migrations/modules/attendance
COPY new-backend/services/grades-service/internal/sql/migrations     /migrations/modules/grades
COPY new-backend/services/meal-service/internal/sql/migrations       /migrations/modules/meal
COPY new-backend/services/payment-service/internal/sql/migrations    /migrations/modules/payment
COPY new-backend/services/notification-service/sql/migrations        /migrations/notification

COPY new-backend/infrastructure/demo-ops/*.sh /scripts/
RUN chmod +x /scripts/*.sh

ENTRYPOINT ["/scripts/ops.sh"]
