-- +goose Up
-- Processed Events Table (Event Deduplication)
CREATE TABLE processed_events (
    event_id UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_processed_events_type ON processed_events(event_type);
CREATE INDEX idx_processed_events_processed_at ON processed_events(processed_at);

-- +goose Down
DROP TABLE IF EXISTS processed_events;
