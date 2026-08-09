-- +goose Up
-- The id of the HTTP request that produced this event, so "which events did
-- this request cause" is answerable in SQL. A separate column rather than a
-- payload field because a JSON key cannot be indexed usefully here.
-- Nullable: worker- and scheduler-driven events have no request behind them.
ALTER TABLE staff.outbox_events ADD COLUMN correlation_id UUID;

CREATE INDEX idx_staff_outbox_correlation
    ON staff.outbox_events(correlation_id)
    WHERE correlation_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS staff.idx_staff_outbox_correlation;
ALTER TABLE staff.outbox_events DROP COLUMN correlation_id;
