-- +goose Up
CREATE TABLE delivery_log (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     TEXT NOT NULL,
    event_type   TEXT NOT NULL,
    channel      TEXT NOT NULL,           -- email, push, sms
    recipient    TEXT NOT NULL,           -- email adresi, telefon, vs.
    template     TEXT NOT NULL,
    status       TEXT NOT NULL,           -- pending, sent, failed
    error        TEXT,
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_delivery_log_event ON delivery_log (event_id);
CREATE INDEX idx_delivery_log_status ON delivery_log (status, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_delivery_log_status;
DROP INDEX IF EXISTS idx_delivery_log_event;
DROP TABLE IF EXISTS delivery_log;
