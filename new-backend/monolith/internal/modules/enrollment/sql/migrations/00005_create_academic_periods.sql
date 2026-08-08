-- +goose Up
-- Local projection of the enrollment deadline, synced from catalog by event.
-- Source: course_catalog.academic_periods where period_type='enrollment'.
-- Only the period event consumer writes here — no request path and no other
-- service touches this table.
-- id is carried over from catalog (no gen_random_uuid default) so a
-- redelivered event upserts the same row instead of forking a second one.
CREATE TABLE enrollment.academic_periods (
    id UUID PRIMARY KEY,
    semester VARCHAR(50) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_enrollment_academic_periods_semester
  ON enrollment.academic_periods(semester);

-- +goose Down
DROP TABLE IF EXISTS enrollment.academic_periods;
