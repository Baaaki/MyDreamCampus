# Architecture Decision Records (ADR)

This directory contains the key architectural decisions made in MyDreamCampus, along with the context and trade-offs behind each one.

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [ADR-001](ADR-001-microservices-architecture.md) | Microservices Architecture over Monolith | Accepted |
| [ADR-002](ADR-002-database-per-service.md) | Database per Service | Accepted |
| [ADR-003](ADR-003-outbox-pattern.md) | Transactional Outbox Pattern for Event Delivery | Accepted |
| [ADR-004](ADR-004-traefik-api-gateway.md) | Traefik as API Gateway | Accepted |
| [ADR-005](ADR-005-sqlc-typesafe-sql.md) | sqlc for Type-Safe SQL over ORM | Accepted |
| [ADR-006](ADR-006-event-driven-communication.md) | RabbitMQ Event-Driven Communication | Accepted |

## What is an ADR?

An Architecture Decision Record captures a significant architectural decision along with its context, reasoning, and consequences. They help answer the question: **"Why did we build it this way?"**

## Format

Each ADR follows this structure:

- **Context**: What problem are we facing?
- **Decision**: What did we choose?
- **Rationale**: Why this option over alternatives?
- **Consequences**: What are the implications?
- **Alternatives Considered**: What else did we evaluate?
