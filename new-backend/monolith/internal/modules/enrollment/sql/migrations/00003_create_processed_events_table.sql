-- +goose Up
CREATE TABLE IF NOT EXISTS enrollment.processed_events (
    event_id UUID PRIMARY KEY,
    event_type VARCHAR(255) NOT NULL,
    processed_at TIMESTAMP DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS enrollment.processed_events;
