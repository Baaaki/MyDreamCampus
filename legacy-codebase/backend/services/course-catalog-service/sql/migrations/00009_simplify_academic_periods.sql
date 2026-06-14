-- +goose Up
-- Remove course_id column from academic_periods (not needed in catalog service)
DROP INDEX IF EXISTS idx_academic_periods_unique_course;
DROP INDEX IF EXISTS idx_academic_periods_unique_global;
ALTER TABLE academic_periods DROP COLUMN IF EXISTS course_id;
CREATE UNIQUE INDEX idx_academic_periods_unique_semester ON academic_periods(semester);

-- +goose Down
ALTER TABLE academic_periods ADD COLUMN course_id UUID NULL;
DROP INDEX IF EXISTS idx_academic_periods_unique_semester;
CREATE UNIQUE INDEX idx_academic_periods_unique_global ON academic_periods(semester) WHERE course_id IS NULL;
CREATE UNIQUE INDEX idx_academic_periods_unique_course ON academic_periods(semester, course_id) WHERE course_id IS NOT NULL;
