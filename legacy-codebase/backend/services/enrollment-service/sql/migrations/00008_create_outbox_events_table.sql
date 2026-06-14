-- +goose Up
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    aggregate_id UUID NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    status VARCHAR(50) DEFAULT 'pending'
);

CREATE INDEX idx_outbox_status ON outbox_events(status) WHERE status = 'pending';
CREATE INDEX idx_outbox_created_at ON outbox_events(created_at);

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
