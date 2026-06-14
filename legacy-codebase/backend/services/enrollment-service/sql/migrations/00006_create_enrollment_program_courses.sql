-- +goose Up
-- Individual courses in a program
CREATE TABLE IF NOT EXISTS enrollment_program_courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES enrollment_programs(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES semester_courses_cache(id),
    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(program_id, course_id)
);

-- +goose Down
DROP TABLE IF EXISTS enrollment_program_courses;
