-- +goose Up
-- ==========================================
-- C. OUTBOX TABLOSU (Event Publishing)
-- ==========================================

CREATE TYPE outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    routing_key VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status outbox_status_enum DEFAULT 'pending',
    retry_count SMALLINT DEFAULT 0,
    max_retries SMALLINT DEFAULT 3,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    error_message TEXT
);

CREATE INDEX idx_outbox_events_pending ON outbox_events(status, created_at) WHERE status = 'pending';

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP TYPE IF EXISTS outbox_status_enum;
