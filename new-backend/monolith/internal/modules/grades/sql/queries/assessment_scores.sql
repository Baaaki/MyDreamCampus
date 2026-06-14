-- name: UpsertAssessmentScore :one
-- Scores are inserted as editable drafts (is_locked=FALSE). An instructor
-- later calls LockScoresByCourseAndSlug to finalize a whole assessment.
-- Already-locked rows cannot be overwritten (admin must unlock first).
INSERT INTO grades.student_assessment_scores (
    registration_id, slug, score, is_absent, graded_by, graded_at, is_locked
) VALUES (
    $1, $2, $3, $4, $5, NOW(), FALSE
)
ON CONFLICT (registration_id, slug) DO UPDATE SET
    score = EXCLUDED.score,
    is_absent = EXCLUDED.is_absent,
    graded_by = EXCLUDED.graded_by,
    graded_at = NOW()
WHERE grades.student_assessment_scores.is_locked = FALSE
RETURNING *;

-- name: GetScoresByRegistration :many
SELECT * FROM grades.student_assessment_scores
WHERE registration_id = $1
ORDER BY graded_at;

-- name: GetScoreByRegistrationAndSlug :one
SELECT * FROM grades.student_assessment_scores
WHERE registration_id = $1 AND slug = $2;

-- name: GetLockedRegistrationsBySlug :many
-- Returns the subset of the given registration IDs whose score at the given
-- slug is already locked. Used by bulk upsert to skip locked entries without
-- per-row roundtrips.
SELECT registration_id
FROM grades.student_assessment_scores
WHERE slug = $1
  AND registration_id = ANY($2::uuid[])
  AND is_locked = TRUE;

-- name: IsScoreLocked :one
SELECT COALESCE(is_locked, FALSE) as is_locked
FROM grades.student_assessment_scores
WHERE registration_id = $1 AND slug = $2;

-- name: UnlockScore :exec
UPDATE grades.student_assessment_scores
SET is_locked = FALSE
WHERE registration_id = $1 AND slug = $2;

-- name: LockScore :exec
UPDATE grades.student_assessment_scores
SET is_locked = TRUE
WHERE registration_id = $1 AND slug = $2;

-- name: CountScoresBySlugAndCourse :one
SELECT COUNT(DISTINCT sa.registration_id)
FROM grades.student_assessment_scores sa
JOIN grades.student_course_registrations r ON sa.registration_id = r.id
WHERE r.course_id = $1 AND sa.slug = $2;

-- name: DeleteScoresByCourse :exec
DELETE FROM grades.student_assessment_scores
WHERE registration_id IN (
    SELECT id FROM grades.student_course_registrations WHERE course_id = $1
);

-- name: CountLockedScoresBySlugAndCourse :one
SELECT COUNT(DISTINCT sa.registration_id)
FROM grades.student_assessment_scores sa
JOIN grades.student_course_registrations r ON sa.registration_id = r.id
WHERE r.course_id = $1 AND sa.slug = $2 AND sa.is_locked = TRUE;

-- name: LockScoresByCourseAndSlug :exec
-- Bulk-lock every score for a given (course, slug). Used by an instructor to
-- finalize an assessment once all students have drafts entered.
UPDATE grades.student_assessment_scores sa
SET is_locked = TRUE
FROM grades.student_course_registrations r
WHERE sa.registration_id = r.id
  AND r.course_id = $1
  AND sa.slug = $2;
