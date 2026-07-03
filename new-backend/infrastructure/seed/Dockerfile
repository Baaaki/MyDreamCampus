# syntax=docker/dockerfile:1
# Tiny toolbox image that seeds demo data via the admin API. curl + jq for the
# HTTP calls, psql only to flip force_password_change off for demo accounts.
FROM alpine:3.20
RUN apk add --no-cache curl jq postgresql-client
COPY seed.sh /seed/seed.sh
COPY seed.sql /seed/seed.sql
COPY data /seed/data
RUN chmod +x /seed/seed.sh
ENTRYPOINT ["/seed/seed.sh"]
