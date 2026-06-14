-- +goose Up
-- Session type enum (theory or lab)
CREATE TYPE session_type_enum AS ENUM ('theory', 'lab');

-- Add session_type to attendance_sessions
ALTER TABLE attendance_sessions
    ADD COLUMN session_type session_type_enum NOT NULL DEFAULT 'theory';

-- Drop old unique constraint and create new one including session_type
ALTER TABLE attendance_sessions
    DROP CONSTRAINT attendance_sessions_course_id_week_number_key;

ALTER TABLE attendance_sessions
    ADD CONSTRAINT attendance_sessions_course_id_week_number_session_type_key
    UNIQUE(course_id, week_number, session_type);

-- Add session_type to attendance_records
ALTER TABLE attendance_records
    ADD COLUMN session_type session_type_enum NOT NULL DEFAULT 'theory';

-- Index for filtering by session_type
CREATE INDEX idx_sessions_session_type ON attendance_sessions(session_type);
CREATE INDEX idx_records_session_type ON attendance_records(session_type);

-- +goose Down
DROP INDEX IF EXISTS idx_records_session_type;
DROP INDEX IF EXISTS idx_sessions_session_type;

ALTER TABLE attendance_records DROP COLUMN IF EXISTS session_type;

ALTER TABLE attendance_sessions
    DROP CONSTRAINT IF EXISTS attendance_sessions_course_id_week_number_session_type_key;

ALTER TABLE attendance_sessions
    ADD CONSTRAINT attendance_sessions_course_id_week_number_key
    UNIQUE(course_id, week_number);

ALTER TABLE attendance_sessions DROP COLUMN IF EXISTS session_type;

DROP TYPE IF EXISTS session_type_enum;
