-- name: CreateRejectionLog :one
INSERT INTO enrollment_rejection_logs (
    original_program_id, student_id, advisor_id, advisor_fullname,
    semester, rejection_reason, rejected_courses, rejected_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, NOW()
)
RETURNING id, original_program_id, student_id, advisor_id, advisor_fullname,
          semester, rejection_reason, rejected_courses, rejected_at;

-- name: GetLatestRejectionByStudentAndSemester :one
SELECT id, original_program_id, student_id, advisor_id, advisor_fullname,
       semester, rejection_reason, rejected_courses, rejected_at
FROM enrollment_rejection_logs
WHERE student_id = $1 AND semester = $2
ORDER BY rejected_at DESC
LIMIT 1;

-- name: GetRejectionsByStudentAndSemester :many
SELECT id, original_program_id, student_id, advisor_id, advisor_fullname,
       semester, rejection_reason, rejected_courses, rejected_at
FROM enrollment_rejection_logs
WHERE student_id = $1
  AND ($2::VARCHAR IS NULL OR $2 = '' OR semester = $2)
ORDER BY rejected_at DESC;

-- name: CountRejectionsByStudentAndSemester :one
SELECT COUNT(*) as total_rejections
FROM enrollment_rejection_logs
WHERE student_id = $1 AND semester = $2;
