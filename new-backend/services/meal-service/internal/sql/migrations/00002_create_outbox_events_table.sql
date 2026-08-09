-- +goose Up
CREATE TYPE meal.outbox_status_enum AS ENUM ('pending', 'published', 'failed');

CREATE TABLE IF NOT EXISTS meal.outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status meal.outbox_status_enum NOT NULL DEFAULT 'pending',
    retry_count SMALLINT NOT NULL DEFAULT 0,
    max_retries SMALLINT NOT NULL DEFAULT 5,
    next_retry_at TIMESTAMPTZ NULL,
    last_error TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_outbox_pending_retry ON meal.outbox_events(next_retry_at) WHERE status = 'pending';
CREATE INDEX idx_outbox_failed ON meal.outbox_events(created_at) WHERE status = 'failed';

CREATE TABLE IF NOT EXISTS meal.processed_events (
    event_id UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_processed_events_type ON meal.processed_events(event_type);
CREATE INDEX idx_processed_events_processed_at ON meal.processed_events(processed_at);

-- +goose Down
DROP TABLE IF EXISTS meal.processed_events;
DROP TABLE IF EXISTS meal.outbox_events;
DROP TYPE IF EXISTS meal.outbox_status_enum;
