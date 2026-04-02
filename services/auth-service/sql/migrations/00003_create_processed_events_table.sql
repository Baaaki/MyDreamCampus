-- +goose Up
CREATE TABLE IF NOT EXISTS processed_events (
    event_id VARCHAR(255) PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_processed_events_type ON processed_events(event_type);
CREATE INDEX idx_processed_events_processed_at ON processed_events(processed_at);

-- +goose Down
DROP INDEX IF EXISTS idx_processed_events_processed_at;
DROP INDEX IF EXISTS idx_processed_events_type;
DROP TABLE IF EXISTS processed_events;
