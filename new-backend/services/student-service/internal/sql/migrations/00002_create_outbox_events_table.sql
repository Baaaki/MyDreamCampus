-- +goose Up
CREATE TYPE student.outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE IF NOT EXISTS student.outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    routing_key VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status student.outbox_status_enum NOT NULL DEFAULT 'pending',
    retry_count SMALLINT NOT NULL DEFAULT 0,
    max_retries SMALLINT NOT NULL DEFAULT 3,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    error_message TEXT
);

CREATE INDEX idx_student_outbox_pending ON student.outbox_events(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_student_outbox_retry ON student.outbox_events(status, retry_count) WHERE status = 'failed';

-- +goose Down
DROP TABLE IF EXISTS student.outbox_events;
DROP TYPE IF EXISTS student.outbox_status_enum;
