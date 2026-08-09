-- +goose Up
-- Backfill teacher_profiles for existing teachers without one. On a fresh
-- monolith DB this is a no-op; preserved for parity with staff-service so
-- production-style data migrations remain in source control.
INSERT INTO staff.teacher_profiles (staff_id)
SELECT id FROM staff.staff
WHERE role = 'teacher'
AND is_active = true
AND id NOT IN (SELECT staff_id FROM staff.teacher_profiles);

-- +goose Down
-- Data migration — leaving rows in place so we do not delete profiles
-- that may have been edited by hand.
SELECT 1;
