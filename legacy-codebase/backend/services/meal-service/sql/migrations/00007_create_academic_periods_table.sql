-- +goose Up
-- Academic periods table for deadline management.
CREATE TABLE IF NOT EXISTS academic_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    semester VARCHAR(50) NOT NULL,
    course_id UUID NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_academic_periods_semester ON academic_periods(semester);
CREATE INDEX IF NOT EXISTS idx_academic_periods_active ON academic_periods(is_active) WHERE is_active = true;
CREATE UNIQUE INDEX IF NOT EXISTS idx_academic_periods_unique_global
    ON academic_periods(semester) WHERE course_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_academic_periods_unique_course
    ON academic_periods(semester, course_id) WHERE course_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS academic_periods;
