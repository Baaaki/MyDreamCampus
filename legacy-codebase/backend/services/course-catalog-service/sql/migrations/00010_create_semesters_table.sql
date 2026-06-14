-- +goose Up

-- Semester status enum
CREATE TYPE semester_status AS ENUM ('planned', 'active', 'completed');

-- Semesters table
CREATE TABLE semesters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    status semester_status NOT NULL DEFAULT 'planned',
    hard_deadline TIMESTAMPTZ NOT NULL,
    activated_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_semesters_status ON semesters(status);
CREATE INDEX idx_semesters_name ON semesters(name);

-- Prevent status regression: completed can never go back
-- This trigger enforces safety even if application layer is bypassed (SQL injection etc.)
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION prevent_semester_reactivation()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status = 'completed' THEN
        RAISE EXCEPTION 'Cannot change status of a completed semester (id: %)', OLD.id;
    END IF;

    IF OLD.status = 'planned' AND NEW.status != 'active' THEN
        RAISE EXCEPTION 'Planned semester can only transition to active (id: %)', OLD.id;
    END IF;

    IF OLD.status = 'active' AND NEW.status != 'completed' THEN
        RAISE EXCEPTION 'Active semester can only transition to completed (id: %)', OLD.id;
    END IF;

    IF OLD.status = 'planned' AND NEW.status = 'active' THEN
        NEW.activated_at = NOW();
    END IF;

    IF OLD.status = 'active' AND NEW.status = 'completed' THEN
        NEW.completed_at = NOW();
    END IF;

    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_semester_status_change
    BEFORE UPDATE OF status ON semesters
    FOR EACH ROW
    EXECUTE FUNCTION prevent_semester_reactivation();

-- +goose Down
DROP TRIGGER IF EXISTS trg_semester_status_change ON semesters;
DROP FUNCTION IF EXISTS prevent_semester_reactivation();
DROP TABLE IF EXISTS semesters;
DROP TYPE IF EXISTS semester_status;
