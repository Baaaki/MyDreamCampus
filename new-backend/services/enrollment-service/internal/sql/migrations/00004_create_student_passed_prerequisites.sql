-- +goose Up
-- Local projection fed by grades' grade.student.prerequisite.passed event.
-- Enrollment validates prerequisites against this table only — it never
-- queries the grades schema, so the module stays microservice-splittable.
-- Keyed by course_code (not course_id): semester course ids are regenerated
-- every term, the code is the stable identity of a course.
CREATE TABLE IF NOT EXISTS enrollment.student_passed_prerequisites (
    student_id  UUID NOT NULL,
    course_id   UUID NOT NULL,
    course_code VARCHAR(50) NOT NULL,
    semester    VARCHAR(50) NOT NULL,
    grade_point VARCHAR(10) NOT NULL,
    synced_at   TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (student_id, course_code)
);

-- +goose Down
DROP TABLE IF EXISTS enrollment.student_passed_prerequisites;
