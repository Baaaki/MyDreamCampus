# ADR-004: Traefik as API Gateway

**Status:** Accepted
**Date:** 2026-01-18
**Decision Makers:** Project Owner

## Context

With 9 backend services, clients (web and mobile) need a single entry point. Without a gateway, the frontend would need to know every service's host and port, handle CORS per service, and manage authentication headers individually.

Options considered:

1. **Nginx** — Traditional reverse proxy, static configuration
2. **Traefik** — Cloud-native reverse proxy with auto-discovery
3. **Kong** — Full API gateway with plugins
4. **Custom BFF (Backend-for-Frontend)** — Go service that aggregates and proxies

## Decision

We use **Traefik v3** as our API gateway, running as a Docker container.

## Rationale

**Why Traefik:**

- **Path-based routing**: `/api/auth/*` → auth-service:8001, `/api/students/*` → student-service:8003. Clean URL structure, single origin for the frontend.
- **Forward Auth middleware**: Traefik natively supports forwarding authentication to an external service. We use this to validate JWT tokens via the auth service's `/api/auth/verify` endpoint — no auth logic duplicated in other services.
- **Middleware chaining**: CORS, rate limiting, authentication, and header injection are configured declaratively in YAML, not in application code.
- **Hot reload**: Traefik watches its config files and applies changes without restart. During development, we can add new routes instantly.
- **Dashboard**: Built-in web UI (port 8080) shows all routes, services, and middleware — invaluable for debugging routing issues.

**Why not Nginx:**

- Nginx requires manual reload for config changes.
- Forward auth requires the `auth_request` module with less intuitive configuration.
- No built-in dashboard.

**Why not Kong:**

- Kong requires its own database (PostgreSQL or Cassandra) — too much overhead for a dev environment.
- Plugin ecosystem is powerful but unnecessary for our needs.

**Why not a custom BFF:**

- A BFF is useful when clients need aggregated data from multiple services in a single request. We may add one later, but for now, the frontend makes parallel requests and the gateway handles cross-cutting concerns.

## Architecture

```
Client (Browser/Mobile)
        │
        ▼
   Traefik (:80)
   ┌─────────────────────┐
   │  CORS Middleware     │
   │  Forward Auth (/verify) │
   │  Rate Limiting       │
   │  Path-based Routing  │
   └─────┬───────────────┘
         │
    ┌────┼────┬────┬────┬────┬────┬────┬────┐
    ▼    ▼    ▼    ▼    ▼    ▼    ▼    ▼    ▼
  auth staff student catalog enroll attend grades meal payment
  :8001 :8002 :8003  :8004  :8005  :8006  :8007 :8008 :50051
```

## Forward Auth Flow

```
1. Client sends request: GET /api/students/123
   Headers: Authorization: Bearer <jwt>

2. Traefik intercepts → forwards to auth-service: GET /api/auth/verify
   (same headers are forwarded)

3. Auth service validates JWT, responds with:
   - 200 OK + headers: X-User-ID, X-User-Role, X-User-Department
   - 401 Unauthorized (invalid/expired token)

4. If 200: Traefik forwards original request to student-service
   with injected headers (X-User-ID, X-User-Role, X-User-Department)

5. Student service reads headers — no JWT parsing needed
```

## Consequences

- All inter-service HTTP calls from clients go through Traefik on port 80.
- Services trust `X-User-*` headers injected by Traefik (since only Traefik can set them).
- CORS is handled once at the gateway level, not per service.
- Adding a new service requires only a new route in `dynamic.yml`.
- Direct service-to-service calls (e.g., student-service calling staff-service) bypass Traefik — they call each other directly via internal ports.
