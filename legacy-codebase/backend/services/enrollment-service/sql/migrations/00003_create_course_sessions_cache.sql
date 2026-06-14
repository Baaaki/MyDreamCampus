-- +goose Up
-- Course schedule sessions cache (slot-based scheduling)
CREATE TABLE IF NOT EXISTS course_sessions_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES semester_courses_cache(id) ON DELETE CASCADE,
    day_of_week day_of_week_enum NOT NULL,
    slot_number INT NOT NULL CHECK (slot_number BETWEEN 1 AND 9),
    synced_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(course_id, day_of_week, slot_number)
);

CREATE INDEX idx_sessions_cache_course ON course_sessions_cache(course_id);
CREATE INDEX idx_sessions_cache_day_slot ON course_sessions_cache(day_of_week, slot_number);

-- +goose Down
DROP TABLE IF EXISTS course_sessions_cache;
