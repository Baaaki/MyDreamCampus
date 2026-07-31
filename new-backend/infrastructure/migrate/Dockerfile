# syntax=docker/dockerfile:1
# One-shot migrator: goose CLI + every module's migration files. Build context
# is the REPO ROOT (Openship's compose builder roots its one shared context at
# the checkout and ignores per-service contexts) — see .dockerignore.

FROM golang:1.26 AS build
ENV CGO_ENABLED=0
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

FROM alpine:3.20
# pg_isready gates the migration run on the server actually accepting
# connections — see entrypoint.sh for why compose's healthcheck isn't enough.
RUN apk add --no-cache postgresql-client
COPY --from=build /go/bin/goose /usr/local/bin/goose

# Monolith modules (one migration dir + one version table each).
COPY new-backend/monolith/internal/modules/auth/sql/migrations           /migrations/modules/auth
COPY new-backend/monolith/internal/modules/staff/sql/migrations          /migrations/modules/staff
COPY new-backend/monolith/internal/modules/student/sql/migrations        /migrations/modules/student
COPY new-backend/monolith/internal/modules/course_catalog/sql/migrations /migrations/modules/course_catalog
COPY new-backend/monolith/internal/modules/enrollment/sql/migrations     /migrations/modules/enrollment
COPY new-backend/monolith/internal/modules/attendance/sql/migrations     /migrations/modules/attendance
COPY new-backend/monolith/internal/modules/grades/sql/migrations         /migrations/modules/grades
COPY new-backend/monolith/internal/modules/meal/sql/migrations           /migrations/modules/meal

# Notification service (separate database).
COPY new-backend/services/notification/sql/migrations                    /migrations/notification

COPY new-backend/infrastructure/migrate/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
