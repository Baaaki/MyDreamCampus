-- name: GetStaffByID :one
SELECT id, email, first_name, last_name, role, department, phone, office_location, is_active, deleted_at, created_at, updated_at
FROM staff
WHERE id = $1 AND is_active = true
LIMIT 1;

-- name: GetStaffByEmail :one
SELECT id, email, first_name, last_name, role, department, phone, office_location, is_active, deleted_at, created_at, updated_at
FROM staff
WHERE email = $1 AND is_active = true
LIMIT 1;

-- name: CreateStaff :one
INSERT INTO staff (email, first_name, last_name, role, department, phone, office_location)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, email, first_name, last_name, role, department, phone, office_location, is_active, deleted_at, created_at, updated_at;

-- name: UpdateStaff :one
UPDATE staff
SET department = COALESCE($2, department),
    phone = COALESCE($3, phone),
    office_location = COALESCE($4, office_location),
    updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING id, email, first_name, last_name, role, department, phone, office_location, is_active, deleted_at, created_at, updated_at;

-- name: SoftDeleteStaff :exec
UPDATE staff
SET is_active = false, deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListStaff :many
SELECT id, email, first_name, last_name, role, department, phone, office_location, is_active, deleted_at, created_at, updated_at
FROM staff
WHERE is_active = true
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountStaff :one
SELECT COUNT(*) FROM staff WHERE is_active = true;