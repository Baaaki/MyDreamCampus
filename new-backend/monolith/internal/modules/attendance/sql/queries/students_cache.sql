-- name: UpsertStudentCache :exec
INSERT INTO attendance.students_view (
    id, student_number, first_name, last_name, email, department, is_active, synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, NOW()
) ON CONFLICT (id) DO UPDATE SET
    student_number = EXCLUDED.student_number,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    email = EXCLUDED.email,
    department = EXCLUDED.department,
    is_active = EXCLUDED.is_active,
    synced_at = NOW();

-- name: GetStudentCacheByID :one
SELECT id, student_number, first_name, last_name, email, department, is_active, synced_at
FROM attendance.students_view
WHERE id = $1
LIMIT 1;

-- name: DeactivateStudentCache :exec
UPDATE attendance.students_view
SET is_active = FALSE, synced_at = NOW()
WHERE id = $1;
