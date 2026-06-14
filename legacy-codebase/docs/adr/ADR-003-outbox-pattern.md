# ADR-003: Transactional Outbox Pattern for Event Delivery

**Status:** Accepted
**Date:** 2026-01-20
**Decision Makers:** Project Owner

## Context

Our microservices need to communicate state changes. When the staff service creates a new instructor, the auth service needs to create login credentials and the student service might need to update advisor references.

The fundamental problem: **how do we reliably publish an event when updating a database?**

### The Dual-Write Problem

A naive approach:

```go
// DANGEROUS — dual write
func (s *StaffService) CreateStaff(staff Staff) error {
    err := s.db.Insert(staff)          // Step 1: write to DB
    if err != nil { return err }

    err = s.rabbit.Publish("staff.created", staff)  // Step 2: publish event
    if err != nil {
        // DB has the record but event was never sent!
        // Other services will never know about this staff member.
        return err
    }
    return nil
}
```

If the application crashes between step 1 and step 2, or if RabbitMQ is temporarily down, the event is lost forever. The database and the message broker are now inconsistent.

## Decision

We use the **Transactional Outbox Pattern**: instead of publishing directly to RabbitMQ, we write the event to an `outbox_events` table in the same database transaction as the business data. A background worker polls the outbox table and publishes events to RabbitMQ.

## How It Works

### Step 1: Business Operation + Outbox Write (Single Transaction)

```go
func (s *StaffService) CreateStaff(staff Staff) error {
    tx, _ := s.db.Begin()

    // Write business data
    tx.Insert("staff", staff)

    // Write event to outbox (same transaction!)
    tx.Insert("outbox_events", OutboxEvent{
        EventType: "staff.created",
        Payload:   toJSON(staff),
        Status:    "pending",
    })

    return tx.Commit()  // Both succeed or both fail — atomic
}
```

### Step 2: Outbox Worker (Background Goroutine)

```
Every 2 seconds:
  1. SELECT * FROM outbox_events WHERE status = 'pending' LIMIT 100
  2. For each event:
     a. Publish to RabbitMQ
     b. UPDATE outbox_events SET status = 'sent'
  3. If publish fails → event stays 'pending', retried next cycle
```

### Step 3: Consumer Side (Idempotency)

```
On message received:
  1. Check processed_events table — have we seen this event ID before?
  2. If yes → ACK and skip (idempotent)
  3. If no → process event, insert into processed_events, ACK
```

## Rationale

**Why outbox over direct publish:**

- **Atomicity**: Business write and event creation are in the same DB transaction. Either both happen or neither does.
- **Durability**: Events are persisted in PostgreSQL. If RabbitMQ is down, events queue up in the outbox and get published when it recovers.
- **Ordering**: Events are inserted with sequential IDs, preserving order within a service.

**Why not Change Data Capture (CDC):**

- CDC (e.g., Debezium reading PostgreSQL WAL) is more sophisticated but requires additional infrastructure (Kafka Connect, Debezium connector).
- The outbox pattern is simpler to implement, understand, and debug.
- For our scale (hundreds of events/day, not millions), outbox is perfectly adequate.

**Why not saga orchestration:**

- We don't have cross-service transactions that need rollback. Our events are notifications ("this happened"), not commands ("do this").
- Choreography (each service reacts to events) is simpler than orchestration (central coordinator) for our use cases.

## Consequences

- Every service that publishes events has an `outbox_events` table and an outbox worker goroutine.
- Every service that consumes events has a `processed_events` table for idempotency.
- Events may be delivered more than once (at-least-once delivery), so consumers must be idempotent.
- There is a small delay (up to the polling interval) between the business write and the event being published.
- Dead Letter Queues (DLQ) capture events that fail processing after max retries.

## Database Schema

```sql
-- Producer side
CREATE TABLE outbox_events (
    id          BIGSERIAL PRIMARY KEY,
    event_type  TEXT NOT NULL,
    payload     JSONB NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',  -- pending | sent
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Consumer side
CREATE TABLE processed_events (
    event_id    TEXT PRIMARY KEY,
    event_type  TEXT NOT NULL,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);
```
