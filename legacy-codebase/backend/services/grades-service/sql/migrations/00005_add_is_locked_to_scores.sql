-- +goose Up
-- Add is_locked column to student_assessment_scores table
-- When a score is submitted, it will be locked by default
ALTER TABLE student_assessment_scores ADD COLUMN is_locked BOOLEAN DEFAULT TRUE;

-- +goose Down
ALTER TABLE student_assessment_scores DROP COLUMN is_locked;
