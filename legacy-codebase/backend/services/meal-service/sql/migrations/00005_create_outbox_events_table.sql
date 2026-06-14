-- +goose Up
CREATE TYPE outbox_status_enum AS ENUM ('pending', 'published', 'failed');

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status outbox_status_enum NOT NULL DEFAULT 'pending',
    retry_count SMALLINT NOT NULL DEFAULT 0,
    max_retries SMALLINT NOT NULL DEFAULT 5,
    next_retry_at TIMESTAMPTZ NULL,
    last_error TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL
);

-- Outbox polling: Pending ve retry zamanı gelmiş eventler
CREATE INDEX idx_outbox_pending_retry
ON outbox_events(next_retry_at)
WHERE status = 'pending';

-- Outbox failed events (monitoring için)
CREATE INDEX idx_outbox_failed
ON outbox_events(created_at)
WHERE status = 'failed';

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP TYPE IF EXISTS outbox_status_enum;
