# syntax=docker/dockerfile:1
# Stage 1 builds the SPA; stage 2 is a Caddy image that serves the built
# assets AND reverse-proxies /api/* to the monolith on the same host, so the
# browser only ever talks to one origin (no CORS, no absolute API base URL).
#
# Build context is the REPO ROOT, not frontend/ — Openship's compose builder
# roots its one shared context at the checkout and ignores per-service
# contexts, so every COPY below is repo-root-relative.

FROM oven/bun:1 AS build
WORKDIR /app
COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY frontend/ .
# Same-origin: empty base URL -> the SPA calls relative /api/... which Caddy
# routes to the monolith. Real API, not mock.
ARG VITE_API_BASE_URL=""
ARG VITE_USE_MOCK_API="false"
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_USE_MOCK_API=$VITE_USE_MOCK_API
RUN bun run build

FROM caddy:2-alpine
COPY --from=build /app/dist /srv
COPY frontend/Caddyfile /etc/caddy/Caddyfile
EXPOSE 80 443
