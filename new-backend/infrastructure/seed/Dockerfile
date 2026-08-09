# syntax=docker/dockerfile:1
# Tiny toolbox image that seeds demo data via the admin API. curl + jq for the
# HTTP calls, psql for the relational half — one script per service database.
# Build context is the REPO ROOT — see .dockerignore for why.
FROM alpine:3.20
RUN apk add --no-cache curl jq postgresql-client
COPY new-backend/infrastructure/seed/seed.sh /seed/seed.sh
COPY new-backend/infrastructure/seed/sql /seed/sql
COPY new-backend/infrastructure/seed/data /seed/data
RUN chmod +x /seed/seed.sh
ENTRYPOINT ["/seed/seed.sh"]
