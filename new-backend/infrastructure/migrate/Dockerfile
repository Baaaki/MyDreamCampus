# syntax=docker/dockerfile:1
# One-shot migrator: goose CLI + every module's migration files. Build context
# is the REPO ROOT — one shared context serves every image, see .dockerignore.

FROM golang:1.27 AS build
ENV CGO_ENABLED=0
# Pinned instead of @latest, so a goose release reaches the migrator only
# through a commit. v3.28.0 is the goose used locally for `make migrate-*`.
ARG GOOSE_VERSION=v3.28.0
# Same caches the service images use, so goose is compiled once per machine
# rather than once per rebuild of this image.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}

FROM alpine:3.24
# pg_isready gates the migration run on the server actually accepting
# connections — see entrypoint.sh for why compose's healthcheck isn't enough.
RUN apk add --no-cache postgresql-client
COPY --from=build /go/bin/goose /usr/local/bin/goose

# One migration dir + one goose version table per service. The
# /migrations/modules/<name> destinations are unchanged on purpose:
# entrypoint.sh maps them to databases by that name, and catalog's dir is
# still called course_catalog because its goose table is.
COPY new-backend/services/auth-service/internal/sql/migrations       /migrations/modules/auth
COPY new-backend/services/staff-service/internal/sql/migrations      /migrations/modules/staff
COPY new-backend/services/student-service/internal/sql/migrations    /migrations/modules/student
COPY new-backend/services/catalog-service/internal/sql/migrations    /migrations/modules/course_catalog
COPY new-backend/services/enrollment-service/internal/sql/migrations /migrations/modules/enrollment
COPY new-backend/services/attendance-service/internal/sql/migrations /migrations/modules/attendance
COPY new-backend/services/grades-service/internal/sql/migrations     /migrations/modules/grades
COPY new-backend/services/meal-service/internal/sql/migrations       /migrations/modules/meal

# Notification service (separate database).
COPY new-backend/services/notification-service/sql/migrations        /migrations/notification

COPY new-backend/infrastructure/migrate/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
