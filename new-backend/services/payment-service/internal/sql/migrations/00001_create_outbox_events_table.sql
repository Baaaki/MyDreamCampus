-- +goose Up
-- Same shape as every other service's outbox, correlation_id included from
-- the start. No processed_events table: payment consumes no events.
CREATE TYPE payment.outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE IF NOT EXISTS payment.outbox_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    event_type VARCHAR(100) NOT NULL,
    routing_key VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status payment.outbox_status_enum NOT NULL DEFAULT 'pending',
    retry_count SMALLINT NOT NULL DEFAULT 0,
    max_retries SMALLINT NOT NULL DEFAULT 3,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP,
    correlation_id UUID
);

CREATE INDEX idx_payment_outbox_pending ON payment.outbox_events(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_payment_outbox_retry ON payment.outbox_events(status, retry_count) WHERE status = 'failed';
CREATE INDEX idx_payment_outbox_correlation
    ON payment.outbox_events(correlation_id)
    WHERE correlation_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS payment.outbox_events;
DROP TYPE IF EXISTS payment.outbox_status_enum;
