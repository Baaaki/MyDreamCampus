-- +goose Up
CREATE TABLE IF NOT EXISTS processed_events (
    event_id UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_processed_events_type ON processed_events(event_type);

-- +goose Down
DROP TABLE IF EXISTS processed_events;
