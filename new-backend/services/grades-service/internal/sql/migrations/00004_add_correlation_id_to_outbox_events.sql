-- +goose Up
-- The id of the HTTP request that produced this event, so "which events did
-- this request cause" is answerable in SQL. A separate column rather than a
-- payload field because a JSON key cannot be indexed usefully here.
-- Nullable: worker- and scheduler-driven events have no request behind them.
ALTER TABLE grades.outbox_events ADD COLUMN correlation_id UUID;

CREATE INDEX idx_grades_outbox_correlation
    ON grades.outbox_events(correlation_id)
    WHERE correlation_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS grades.idx_grades_outbox_correlation;
ALTER TABLE grades.outbox_events DROP COLUMN correlation_id;
