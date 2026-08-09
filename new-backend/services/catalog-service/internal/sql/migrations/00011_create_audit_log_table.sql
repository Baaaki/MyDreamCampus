-- +goose Up

CREATE TABLE course_catalog.audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    service VARCHAR(50) NOT NULL,
    actor_id UUID NOT NULL,
    actor_role VARCHAR(20) NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    details JSONB
);

CREATE INDEX idx_audit_log_timestamp ON course_catalog.audit_log(timestamp DESC);
CREATE INDEX idx_audit_log_service ON course_catalog.audit_log(service);
CREATE INDEX idx_audit_log_action ON course_catalog.audit_log(action);
CREATE INDEX idx_audit_log_actor ON course_catalog.audit_log(actor_id);

-- Immutability: prevent UPDATE and DELETE on audit log
-- Even if application layer is bypassed, audit records are protected
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit log entries cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_audit_no_update
    BEFORE UPDATE OR DELETE ON course_catalog.audit_log
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

-- +goose Down
DROP TRIGGER IF EXISTS trg_audit_no_update ON course_catalog.audit_log;
DROP FUNCTION IF EXISTS prevent_audit_modification();
DROP TABLE IF EXISTS course_catalog.audit_log;
