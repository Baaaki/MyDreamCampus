-- +goose Up
-- Catalog is the source of truth for every service's academic period: one row
-- per consuming service instead of one row per semester. Consumers keep an
-- event-fed projection in their own schema, so nobody reads this table across
-- a service boundary anymore.
ALTER TABLE course_catalog.academic_periods
  ADD COLUMN period_type VARCHAR(20) NOT NULL DEFAULT 'catalog';

-- One row per (semester, type) — the old constraint allowed one row per
-- semester, which is exactly what this change replaces.
DROP INDEX IF EXISTS course_catalog.idx_academic_periods_unique_semester;
CREATE UNIQUE INDEX idx_academic_periods_unique_semester_type
  ON course_catalog.academic_periods(semester, period_type);

ALTER TABLE course_catalog.academic_periods
  ADD CONSTRAINT chk_period_type
  CHECK (period_type IN ('catalog','enrollment','grading','attendance'));

-- +goose Down
ALTER TABLE course_catalog.academic_periods DROP CONSTRAINT IF EXISTS chk_period_type;
DROP INDEX IF EXISTS course_catalog.idx_academic_periods_unique_semester_type;
-- The consumer rows only exist because of this migration, and one-row-per-
-- semester cannot be restored while they are present.
DELETE FROM course_catalog.academic_periods WHERE period_type <> 'catalog';
CREATE UNIQUE INDEX idx_academic_periods_unique_semester
  ON course_catalog.academic_periods(semester);
ALTER TABLE course_catalog.academic_periods DROP COLUMN period_type;
