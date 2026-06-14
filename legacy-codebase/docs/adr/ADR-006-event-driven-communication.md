# ADR-006: RabbitMQ Event-Driven Communication

**Status:** Accepted
**Date:** 2026-01-20
**Decision Makers:** Project Owner

## Context

Microservices need to communicate state changes. When a staff member is created, the auth service needs to create credentials. When a student enrolls, the attendance service needs to know.

Options:

1. **Synchronous HTTP calls** — Service A calls Service B's API directly
2. **RabbitMQ (AMQP)** — Asynchronous message-based communication
3. **Apache Kafka** — Distributed event streaming platform
4. **Redis Pub/Sub** — Lightweight publish-subscribe

## Decision

We use **RabbitMQ** with **topic exchanges** for asynchronous inter-service communication.

## Rationale

**Why asynchronous events over synchronous HTTP:**

- **Temporal decoupling**: If the attendance service is down when a student enrolls, the enrollment still succeeds. The attendance service processes the event when it comes back online.
- **No cascading failures**: A slow grades service doesn't make the enrollment service slow.
- **Fan-out**: One event can be consumed by multiple services. `staff.created` is consumed by both auth-service (to create credentials) and student-service (to update advisor references).

**Why RabbitMQ over Kafka:**

- **Simpler operational model**: RabbitMQ is a message broker — messages are delivered and acknowledged. Kafka is a distributed log — consumers track offsets, partitions need management.
- **Message acknowledgment**: RabbitMQ supports per-message ACK/NACK with redelivery. Perfect for our "process and confirm" pattern.
- **Sufficient scale**: We have hundreds of events per day, not millions. Kafka's strengths (high throughput, log compaction, replay) are unnecessary.
- **Dead Letter Queues**: Native DLQ support for failed messages.
- **Lower resource usage**: Single RabbitMQ instance vs. Kafka + ZooKeeper cluster.

**Why RabbitMQ over Redis Pub/Sub:**

- **Durability**: Redis Pub/Sub is fire-and-forget. If no consumer is listening when a message is published, it's lost. RabbitMQ queues persist messages until consumed.
- **Acknowledgment**: Redis has no ACK mechanism. RabbitMQ ensures messages aren't lost if a consumer crashes mid-processing.

## Event Flow Architecture

```
Staff Service                    RabbitMQ                      Auth Service
─────────────                    ────────                      ────────────

1. Create staff record
2. Write to outbox table
        │
   Outbox Worker
3. Poll outbox ──────────►  4. Publish to
                                topic exchange
                                "staff.created"
                                     │
                                     ├────►  5. auth-service queue
                                     │           │
                                     │       6. Consumer receives
                                     │       7. Create user credentials
                                     │       8. ACK message
                                     │
                                     └────►  student-service queue
                                                 │
                                             (fan-out to
                                              other consumers)
```

## Event Naming Convention

```
{domain}.{entity}.{action}

Examples:
  staff.created
  staff.updated
  student.created
  course.semester.created
  course.semester.updated
  enrollment.enrolled
  enrollment.dropped
```

## Consumer Idempotency

Every consumer checks a `processed_events` table before processing:

```
1. Receive message with event_id
2. SELECT FROM processed_events WHERE event_id = ?
3. If found → already processed → ACK and skip
4. If not found → process event → INSERT into processed_events → ACK
```

This handles the at-least-once delivery guarantee of RabbitMQ.

## Dead Letter Queue (DLQ)

Messages that fail processing after max retries (default: 3) are routed to a DLQ:

```
Queue: staff_events_auth
  └─ on failure (3x) ─► DLQ: staff_events_auth.dlq
```

DLQ messages can be inspected via RabbitMQ Management UI (port 15672) and manually replayed.

## Consequences

- All services include RabbitMQ connection setup in `cmd/main.go`.
- Producer services have outbox workers (`internal/worker/outbox_worker.go`).
- Consumer services have event consumers (`internal/worker/event_consumer.go`).
- RabbitMQ Management UI is available at `localhost:15672` for debugging.
- Event schemas are implicitly defined by JSON payloads — no schema registry.
