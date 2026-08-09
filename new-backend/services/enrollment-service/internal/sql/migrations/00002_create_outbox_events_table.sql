-- +goose Up
CREATE TYPE enrollment.outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE IF NOT EXISTS enrollment.outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(255) NOT NULL,
    routing_key VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status enrollment.outbox_status_enum DEFAULT 'pending',
    retry_count SMALLINT DEFAULT 0,
    max_retries SMALLINT DEFAULT 3,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP
);

CREATE INDEX idx_outbox_events_status ON enrollment.outbox_events(status);
CREATE INDEX idx_outbox_events_created_at ON enrollment.outbox_events(created_at);

-- +goose Down
DROP TABLE IF EXISTS enrollment.outbox_events;
DROP TYPE IF EXISTS enrollment.outbox_status_enum;
