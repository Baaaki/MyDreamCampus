-- +goose Up
-- Academic periods table for deadline management.
-- course_id NULL = global period for the semester.
-- course_id NOT NULL = course-specific override (e.g. deadline extension).
CREATE TABLE course_catalog.academic_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    semester VARCHAR(50) NOT NULL,
    course_id UUID NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_academic_periods_semester ON course_catalog.academic_periods(semester);
CREATE INDEX idx_academic_periods_active ON course_catalog.academic_periods(is_active) WHERE is_active = true;

-- Partial unique indexes to handle NULL course_id correctly.
-- PostgreSQL does NOT treat two NULLs as equal in a standard UNIQUE constraint,
-- so we use partial indexes instead:
-- 1. Only one global period (course_id IS NULL) per semester.
CREATE UNIQUE INDEX idx_academic_periods_unique_global
    ON course_catalog.academic_periods(semester) WHERE course_id IS NULL;
-- 2. Only one course-specific period per (semester, course_id) pair.
CREATE UNIQUE INDEX idx_academic_periods_unique_course
    ON course_catalog.academic_periods(semester, course_id) WHERE course_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS course_catalog.academic_periods;
