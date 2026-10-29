-- +goose Up
-- When the student's program was approved, from the approval event. A
-- student is held only to the sessions run after it: one approved in week 5
-- was counted absent from weeks 1-4, when they could not scan at all. NULL
-- (rows written before this column, and the seed) means from the start.
ALTER TABLE attendance.enrollments_view ADD COLUMN enrolled_at TIMESTAMP;

-- +goose Down
ALTER TABLE attendance.enrollments_view DROP COLUMN IF EXISTS enrolled_at;
