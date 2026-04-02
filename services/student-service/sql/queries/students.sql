-- name: GetStudentByID :one
SELECT id, student_number, first_name, last_name, email, faculty, department,
       enrollment_year, class_level, advisor_id, status, is_active, deleted_at,
       created_at, updated_at
FROM students
WHERE id = $1 AND is_active = true
LIMIT 1;

-- name: GetStudentByEmail :one
SELECT id, student_number, first_name, last_name, email, faculty, department,
       enrollment_year, class_level, advisor_id, status, is_active, deleted_at,
       created_at, updated_at
FROM students
WHERE email = $1 AND is_active = true
LIMIT 1;

-- name: GetStudentByNumber :one
SELECT id, student_number, first_name, last_name, email, faculty, department,
       enrollment_year, class_level, advisor_id, status, is_active, deleted_at,
       created_at, updated_at
FROM students
WHERE student_number = $1 AND is_active = true
LIMIT 1;

-- name: CreateStudent :one
INSERT INTO students (
    student_number, first_name, last_name, email, faculty, department,
    enrollment_year, class_level, advisor_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, student_number, first_name, last_name, email, faculty, department,
          enrollment_year, class_level, advisor_id, status, is_active, deleted_at,
          created_at, updated_at;

-- name: UpdateStudent :one
UPDATE students
SET class_level = COALESCE($2, class_level),
    advisor_id = COALESCE($3, advisor_id),
    status = COALESCE($4, status),
    updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING id, student_number, first_name, last_name, email, faculty, department,
          enrollment_year, class_level, advisor_id, status, is_active, deleted_at,
          created_at, updated_at;

-- name: SoftDeleteStudent :exec
UPDATE students
SET is_active = false, deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListStudents :many
SELECT id, student_number, first_name, last_name, email, faculty, department,
       enrollment_year, class_level, advisor_id, status, is_active, deleted_at,
       created_at, updated_at
FROM students
WHERE is_active = true
  AND (sqlc.narg('department')::TEXT IS NULL OR department = sqlc.narg('department'))
  AND (sqlc.narg('class_level')::SMALLINT IS NULL OR class_level = sqlc.narg('class_level'))
  AND (sqlc.narg('status')::TEXT IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('advisor_id')::UUID IS NULL OR advisor_id = sqlc.narg('advisor_id'))
ORDER BY
  CASE WHEN sqlc.arg('sort_by') = 'student_number' AND sqlc.arg('sort_order') = 'asc' THEN student_number END ASC,
  CASE WHEN sqlc.arg('sort_by') = 'student_number' AND sqlc.arg('sort_order') = 'desc' THEN student_number END DESC,
  CASE WHEN sqlc.arg('sort_by') = 'last_name' AND sqlc.arg('sort_order') = 'asc' THEN last_name END ASC,
  CASE WHEN sqlc.arg('sort_by') = 'last_name' AND sqlc.arg('sort_order') = 'desc' THEN last_name END DESC,
  CASE WHEN sqlc.arg('sort_by') = 'enrollment_year' AND sqlc.arg('sort_order') = 'asc' THEN enrollment_year END ASC,
  CASE WHEN sqlc.arg('sort_by') = 'enrollment_year' AND sqlc.arg('sort_order') = 'desc' THEN enrollment_year END DESC,
  CASE WHEN sqlc.arg('sort_by') = 'class_level' AND sqlc.arg('sort_order') = 'asc' THEN class_level END ASC,
  CASE WHEN sqlc.arg('sort_by') = 'class_level' AND sqlc.arg('sort_order') = 'desc' THEN class_level END DESC,
  CASE WHEN sqlc.arg('sort_by') = 'created_at' AND sqlc.arg('sort_order') = 'asc' THEN created_at END ASC,
  CASE WHEN sqlc.arg('sort_by') = 'created_at' AND sqlc.arg('sort_order') = 'desc' THEN created_at END DESC,
  created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListStudentsByAdvisor :many
SELECT id, student_number, first_name, last_name, email, faculty, department,
       enrollment_year, class_level, advisor_id, status, is_active, deleted_at,
       created_at, updated_at
FROM students
WHERE advisor_id = $1 AND is_active = true AND status = 'active'
ORDER BY class_level ASC, last_name ASC;

-- name: ListOrphanedStudents :many
SELECT id, student_number, first_name, last_name, email, faculty, department,
       enrollment_year, class_level, advisor_id, status, is_active, deleted_at,
       created_at, updated_at
FROM students
WHERE advisor_id IS NULL AND is_active = true
ORDER BY department ASC, class_level ASC, last_name ASC
LIMIT $1 OFFSET $2;

-- name: CountStudents :one
SELECT COUNT(*) FROM students WHERE is_active = true;

-- name: CountOrphanedStudents :one
SELECT COUNT(*) FROM students WHERE advisor_id IS NULL AND is_active = true;

-- name: BulkAssignAdvisor :exec
UPDATE students
SET advisor_id = $2, updated_at = NOW()
WHERE id = ANY($1::UUID[]) AND is_active = true;

-- name: UnassignAdvisorByStaffID :exec
UPDATE students
SET advisor_id = NULL, updated_at = NOW()
WHERE advisor_id = $1;

-- name: SearchStudents :many
SELECT id, student_number, first_name, last_name, email, faculty, department,
       enrollment_year, class_level, advisor_id, status, is_active, deleted_at,
       created_at, updated_at
FROM students
WHERE is_active = true
  AND (sqlc.narg('query')::TEXT IS NULL OR to_tsvector('english', first_name || ' ' || last_name || ' ' || student_number) @@ plainto_tsquery('english', sqlc.narg('query')))
  AND (sqlc.narg('department')::TEXT IS NULL OR department = sqlc.narg('department'))
  AND (sqlc.narg('class_level')::SMALLINT IS NULL OR class_level = sqlc.narg('class_level'))
  AND (sqlc.narg('status')::TEXT IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('advisor_id')::UUID IS NULL OR advisor_id = sqlc.narg('advisor_id'))
ORDER BY last_name ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListStudentsByDepartment :many
SELECT id, student_number, first_name, last_name, email, faculty, department,
       enrollment_year, class_level, advisor_id, status, is_active, deleted_at,
       created_at, updated_at
FROM students
WHERE department = $1 AND is_active = true
ORDER BY class_level ASC, last_name ASC;
