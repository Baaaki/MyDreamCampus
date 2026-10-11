-- +goose Up
-- One row per course whose attendance was finalized. A course id belongs to
-- a single semester, so it alone keys the row: a second finalize finds it and
-- is refused instead of queuing every attendance.semester.failed event again.
-- result keeps the summary the teacher saw, so the page can show it later.
CREATE TABLE attendance.course_finalizations (
    course_id UUID PRIMARY KEY REFERENCES attendance.courses_view(id) ON DELETE CASCADE,
    semester VARCHAR(50) NOT NULL,
    finalized_by UUID NOT NULL,
    finalized_at TIMESTAMPTZ NOT NULL,
    result JSONB NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS attendance.course_finalizations;
