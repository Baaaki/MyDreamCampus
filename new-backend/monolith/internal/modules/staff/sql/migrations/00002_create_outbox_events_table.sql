-- +goose Up
-- Plan section 5.2 — final outbox shape (UUID id + status enum + retry).
-- The legacy SERIAL → UUID upgrade dance is collapsed into this single
-- create migration on the clean monolith database.
CREATE TYPE staff.outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE IF NOT EXISTS staff.outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    routing_key VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status staff.outbox_status_enum NOT NULL DEFAULT 'pending',
    retry_count SMALLINT NOT NULL DEFAULT 0,
    max_retries SMALLINT NOT NULL DEFAULT 3,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP
);

CREATE INDEX idx_staff_outbox_pending ON staff.outbox_events(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_staff_outbox_retry ON staff.outbox_events(status, retry_count) WHERE status = 'failed';

-- +goose Down
DROP TABLE IF EXISTS staff.outbox_events;
DROP TYPE IF EXISTS staff.outbox_status_enum;
