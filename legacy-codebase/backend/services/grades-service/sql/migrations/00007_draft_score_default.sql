-- +goose Up
-- Scores are now entered as drafts (is_locked=FALSE) by default.
-- The instructor explicitly locks an entire assessment slug when ready;
-- once every assessment is locked, the course auto-finalizes.
ALTER TABLE student_assessment_scores ALTER COLUMN is_locked SET DEFAULT FALSE;

-- +goose Down
ALTER TABLE student_assessment_scores ALTER COLUMN is_locked SET DEFAULT TRUE;
