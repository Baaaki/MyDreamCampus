-- name: ListAdminStaff :many
SELECT * FROM staff.admin_staff
WHERE is_active = true
  AND (sqlc.narg('faculty')::text IS NULL OR faculty = sqlc.narg('faculty'))
ORDER BY last_name, first_name;

-- name: GetAdminStaffByID :one
SELECT * FROM staff.admin_staff
WHERE id = $1 AND is_active = true
LIMIT 1;

-- name: CreateAdminStaff :one
INSERT INTO staff.admin_staff (
    email, title, first_name, last_name, faculty, department, phone,
    profile_image_url, position, job_description, responsibilities,
    working_hours, office_location, start_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: UpdateAdminStaff :one
UPDATE staff.admin_staff
SET email = $2,
    title = $3,
    first_name = $4,
    last_name = $5,
    faculty = $6,
    department = $7,
    phone = $8,
    profile_image_url = $9,
    position = $10,
    job_description = $11,
    responsibilities = $12,
    working_hours = $13,
    office_location = $14,
    start_date = $15,
    updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING *;
