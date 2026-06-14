-- +goose Up
-- Create teacher_profiles for existing teachers who don't have one
INSERT INTO teacher_profiles (staff_id)
SELECT id FROM staff
WHERE role = 'teacher'
AND is_active = true
AND id NOT IN (SELECT staff_id FROM teacher_profiles);

-- +goose Down
-- This is a data migration, down migration would delete the seeded profiles
-- But we don't want to delete profiles that might have been updated
-- So we do nothing on down
