-- +goose Up
CREATE TABLE IF NOT EXISTS course_schedule_sessions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    semester_course_id UUID NOT NULL REFERENCES semester_courses(id) ON DELETE CASCADE,
    day_of_week day_of_week_enum NOT NULL,
    slot_number SMALLINT NOT NULL CHECK (slot_number BETWEEN 1 AND 9),
    created_at TIMESTAMP DEFAULT NOW(),

    -- Prevent duplicate: Same course, same day, same slot
    UNIQUE(semester_course_id, day_of_week, slot_number)
);

CREATE INDEX idx_schedule_sessions_course ON course_schedule_sessions(semester_course_id);
CREATE INDEX idx_schedule_sessions_day_slot ON course_schedule_sessions(day_of_week, slot_number);

-- +goose Down
DROP TABLE IF EXISTS course_schedule_sessions;
