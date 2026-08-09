-- +goose Up
CREATE TYPE course_catalog.outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE IF NOT EXISTS course_catalog.outbox_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    event_type VARCHAR(100) NOT NULL,           -- 'course.semester.created', 'course.semester.updated', 'course.semester.deleted'
    routing_key VARCHAR(100) NOT NULL,          -- RabbitMQ routing key
    payload JSONB NOT NULL,                     -- Event data (JSON)
    status course_catalog.outbox_status_enum DEFAULT 'pending',
    retry_count SMALLINT DEFAULT 0,
    max_retries SMALLINT DEFAULT 3,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    error_message TEXT                          -- Son hata mesajı (debug için)
);

CREATE INDEX idx_outbox_events_pending ON course_catalog.outbox_events(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_outbox_events_retry ON course_catalog.outbox_events(status, retry_count) WHERE status = 'failed';

-- +goose Down
DROP TABLE IF EXISTS course_catalog.outbox_events;
DROP TYPE IF EXISTS course_catalog.outbox_status_enum;
