-- +goose Up
-- Replace academic_periods with closed_days (meal service uses holidays, not deadlines)
DROP TABLE IF EXISTS academic_periods;

CREATE TABLE closed_days (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL UNIQUE,
    reason VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_closed_days_date ON closed_days(date);

-- +goose Down
DROP TABLE IF EXISTS closed_days;

CREATE TABLE academic_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    semester VARCHAR(50) NOT NULL,
    course_id UUID NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_academic_periods_semester ON academic_periods(semester);
CREATE INDEX idx_academic_periods_active ON academic_periods(is_active) WHERE is_active = true;
CREATE UNIQUE INDEX idx_academic_periods_unique_global ON academic_periods(semester) WHERE course_id IS NULL;
CREATE UNIQUE INDEX idx_academic_periods_unique_course ON academic_periods(semester, course_id) WHERE course_id IS NOT NULL;
