-- +goose Up
-- Enum for schedule session type (theory vs lab)
CREATE TYPE schedule_session_type_enum AS ENUM (
    'theory',  -- Teorik ders
    'lab'      -- Uygulama/Lab dersi
);

-- Add session_type column to course_schedule_sessions
ALTER TABLE course_schedule_sessions
    ADD COLUMN session_type schedule_session_type_enum NOT NULL DEFAULT 'theory';

-- Drop old unique constraint and create new one that includes session_type
-- Old: UNIQUE(semester_course_id, day_of_week, slot_number)
-- New: same slot can be used for different session types (unlikely but structurally correct)
-- Actually we keep the same constraint - same course can't have both theory and lab at the same slot
-- The existing constraint already handles this correctly

-- Add index for filtering by session_type
CREATE INDEX idx_schedule_sessions_type ON course_schedule_sessions(session_type);

-- +goose Down
DROP INDEX IF EXISTS idx_schedule_sessions_type;
ALTER TABLE course_schedule_sessions DROP COLUMN IF EXISTS session_type;
DROP TYPE IF EXISTS schedule_session_type_enum;
