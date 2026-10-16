-- +goose Up
-- CountActiveCourseEnrollments (the seat count every enrollment checks)
-- filters on course_id alone. UNIQUE(program_id, course_id) leads with
-- program_id, so that lookup had no index it could seek on.
CREATE INDEX IF NOT EXISTS idx_program_courses_course
    ON enrollment.enrollment_program_courses(course_id);

-- +goose Down
DROP INDEX IF EXISTS enrollment.idx_program_courses_course;
