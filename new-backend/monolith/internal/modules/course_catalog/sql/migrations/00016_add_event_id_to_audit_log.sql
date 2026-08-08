-- +goose Up
-- Audit entries from grades and meal now arrive as events instead of an
-- in-process write. The event id is the dedup key: a redelivered message
-- must not add a second row. Nullable because catalog's own writes have no
-- event behind them.
ALTER TABLE course_catalog.audit_log ADD COLUMN event_id UUID;

CREATE UNIQUE INDEX idx_audit_log_event_id
    ON course_catalog.audit_log(event_id)
    WHERE event_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS course_catalog.idx_audit_log_event_id;
ALTER TABLE course_catalog.audit_log DROP COLUMN event_id;
