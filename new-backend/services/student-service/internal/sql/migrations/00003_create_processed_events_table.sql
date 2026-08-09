-- +goose Up
CREATE TABLE IF NOT EXISTS student.processed_events (
    event_id VARCHAR(255) PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_student_processed_events_type ON student.processed_events(event_type);
CREATE INDEX idx_student_processed_events_processed_at ON student.processed_events(processed_at);

-- +goose Down
DROP TABLE IF EXISTS student.processed_events;
