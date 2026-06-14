# ADR-002: Database per Service

**Status:** Accepted
**Date:** 2026-01-15
**Decision Makers:** Project Owner

## Context

With 9 microservices, we need a data storage strategy. The main options:

1. **Shared database** — All services read/write a single PostgreSQL instance with a shared schema
2. **Shared database, separate schemas** — One PostgreSQL instance, each service has its own schema
3. **Database per service** — Each service has its own PostgreSQL instance

## Decision

Each service gets its own **dedicated PostgreSQL instance** (9 total), running as separate Docker containers.

## Rationale

**Why database per service:**

- **True encapsulation**: No service can accidentally (or intentionally) query another service's tables. The boundary is enforced at the network level, not by convention.
- **Independent migrations**: Each service manages its own schema evolution with Goose. The enrollment service can run a migration without touching the grades database.
- **Failure isolation**: If `postgres-meal` crashes or runs a heavy migration, `postgres-auth` is unaffected.
- **Realistic production modeling**: In a real-world microservices deployment, services would have separate databases (often on separate servers/clusters). Using separate instances in development means our code never accidentally relies on cross-database joins.

**Trade-offs accepted:**

- **Resource usage**: 9 PostgreSQL containers consume more memory than one. In development this is ~200MB per container. Acceptable for a dev machine.
- **No cross-service joins**: Need a student's enrollment and grades in one view? The frontend calls both services separately, or a BFF (Backend-for-Frontend) aggregates them.
- **No cross-service foreign keys**: We store references (e.g., `student_id` in the enrollment DB) but cannot enforce referential integrity across databases. Consistency depends on event-driven synchronization.
- **Data duplication**: Some data (e.g., student names, course names) is replicated across services via events. This is intentional — each service stores the data it needs to operate independently.

## Consequences

- The `docker-compose.yml` defines 9 separate PostgreSQL services with unique ports (5432-5440).
- Each service's `config.go` points to its own database connection string.
- Cross-service data needs are satisfied through:
  - Synchronous HTTP calls (rare, only when real-time accuracy is needed)
  - Asynchronous events via RabbitMQ (preferred — eventual consistency)
- Data migration scripts are per-service in `sql/migrations/` directories.

## Port Mapping

| Service | DB Container | Port |
|---------|-------------|------|
| auth | postgres-auth | 5432 |
| staff | postgres-staff | 5433 |
| student | postgres-student | 5434 |
| catalog | postgres-catalog | 5435 |
| enrollment | postgres-enrollment | 5436 |
| attendance | postgres-attendance | 5437 |
| grades | postgres-grades | 5438 |
| meal | postgres-meal | 5439 |
