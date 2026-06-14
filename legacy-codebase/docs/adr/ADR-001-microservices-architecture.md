# ADR-001: Microservices Architecture over Monolith

**Status:** Accepted
**Date:** 2026-01-15
**Decision Makers:** Project Owner

## Context

MyDreamCampus is a university management system covering authentication, staff management, student records, course catalog, enrollment, attendance, grades, and meal services. The system needs to serve web and mobile clients simultaneously.

Two architectural approaches were considered:

1. **Monolith** — Single Go binary, single database, all domains in one codebase
2. **Microservices** — Independent services per domain, each with its own database

## Decision

We chose **microservices architecture** with 9 independent Go services.

## Rationale

**Why microservices:**

- **Domain isolation**: University domains (enrollment vs. meals vs. grades) have fundamentally different data models, access patterns, and change frequencies. Enrollment changes every semester; meal menus change daily; grade calculation rules rarely change.
- **Independent deployment**: A bug fix in the meal service should not require redeploying the auth service.
- **Technology flexibility**: Each service can choose its own database schema, migration strategy, and internal patterns without affecting others.
- **Learning objective**: Building a real microservices system teaches distributed systems concepts (eventual consistency, service discovery, API gateway patterns) that a monolith cannot.

**Known trade-offs we accepted:**

- **Operational complexity**: 9 databases, 9 processes, message broker, API gateway — significantly more infrastructure than a monolith.
- **Data consistency**: No cross-service transactions. We must handle eventual consistency with the outbox pattern.
- **Network overhead**: Inter-service communication adds latency compared to in-process function calls.
- **Code duplication**: Some DTOs and validation logic exists in multiple services, even with a shared package.

**Why not start with a monolith and split later (the "monolith-first" approach):**

- This is a greenfield educational project, not a startup racing to market. The cost of premature decomposition is learning infrastructure patterns; the cost of a monolith is missing those patterns entirely.
- The domain boundaries (auth, enrollment, grades, etc.) are well-understood and stable — this isn't a domain where we'd discover boundaries through iteration.

## Consequences

- Every new feature must consider which service owns it and how cross-service data flows work.
- Infrastructure setup (Docker Compose, Traefik, RabbitMQ) is a prerequisite before any business logic development.
- Developers must understand event-driven patterns and eventual consistency.
- Debugging requires log aggregation (Loki + Grafana) since logs are distributed across services.

## Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| Monolith | Simple deployment, easy debugging, single DB transactions | All-or-nothing deploys, tight coupling, less to learn about distributed systems |
| Modular Monolith | Single deployment with clear module boundaries | Still shares DB, module boundaries can erode over time |
| **Microservices** ✅ | True isolation, independent scaling, real distributed systems experience | Operational complexity, eventual consistency challenges |
