-- +goose Up
-- ==========================================
-- C. OUTBOX & PROCESSED EVENTS
-- ==========================================

CREATE TYPE attendance.outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE attendance.outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    routing_key VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status attendance.outbox_status_enum DEFAULT 'pending',
    retry_count SMALLINT DEFAULT 0,
    max_retries SMALLINT DEFAULT 3,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    error_message TEXT
);

CREATE INDEX idx_outbox_events_pending ON attendance.outbox_events(status, created_at) WHERE status = 'pending';

CREATE TABLE attendance.processed_events (
    event_id UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_processed_events_type ON attendance.processed_events(event_type);
CREATE INDEX idx_processed_events_processed_at ON attendance.processed_events(processed_at);

-- +goose Down
DROP TABLE IF EXISTS attendance.processed_events;
DROP TABLE IF EXISTS attendance.outbox_events;
DROP TYPE IF EXISTS attendance.outbox_status_enum;
