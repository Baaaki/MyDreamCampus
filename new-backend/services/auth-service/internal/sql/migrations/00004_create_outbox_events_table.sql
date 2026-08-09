-- +goose Up
-- Auth module outbox table for event publishing (user.registered, user.password_reset_requested)

CREATE TYPE auth.outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE IF NOT EXISTS auth.outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    routing_key VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status auth.outbox_status_enum NOT NULL DEFAULT 'pending',
    retry_count SMALLINT NOT NULL DEFAULT 0,
    max_retries SMALLINT NOT NULL DEFAULT 3,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP
);

CREATE INDEX idx_auth_outbox_pending ON auth.outbox_events(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_auth_outbox_retry ON auth.outbox_events(status, retry_count) WHERE status = 'failed';

-- +goose Down
DROP TABLE IF EXISTS auth.outbox_events;
DROP TYPE IF EXISTS auth.outbox_status_enum;
