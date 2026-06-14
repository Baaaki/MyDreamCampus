-- +goose Up

-- Add department column to semester_courses (cached from course_catalog for uniqueness)
ALTER TABLE semester_courses ADD COLUMN department VARCHAR(100) NOT NULL DEFAULT '';

-- Backfill department from course_catalog for existing rows
UPDATE semester_courses sc
SET department = cc.department
FROM course_catalog cc
WHERE sc.course_code = cc.course_code;

-- Drop old unique constraint and create new one
ALTER TABLE semester_courses DROP CONSTRAINT semester_courses_semester_course_code_key;
ALTER TABLE semester_courses ADD CONSTRAINT semester_courses_semester_course_code_department_key UNIQUE(semester, course_code, department);

-- Remove default after backfill
ALTER TABLE semester_courses ALTER COLUMN department DROP DEFAULT;

CREATE INDEX idx_semester_courses_department ON semester_courses(department);

-- +goose Down
DROP INDEX IF EXISTS idx_semester_courses_department;
ALTER TABLE semester_courses DROP CONSTRAINT semester_courses_semester_course_code_department_key;
ALTER TABLE semester_courses ADD CONSTRAINT semester_courses_semester_course_code_key UNIQUE(semester, course_code);
ALTER TABLE semester_courses DROP COLUMN department;
