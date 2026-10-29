-- +goose Up
-- The enrollment program a registration came from. A student may change an
-- approved program while the window is open: its cancelled event removes
-- only that program's registrations, so a later program's approval that is
-- consumed first keeps the courses it shares with the old one. NULL (rows
-- written before this column, and the seed) is matched by course alone.
ALTER TABLE grades.student_course_registrations ADD COLUMN program_id UUID;

-- +goose Down
ALTER TABLE grades.student_course_registrations DROP COLUMN IF EXISTS program_id;
