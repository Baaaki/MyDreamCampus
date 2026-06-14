-- name: GetSessionsByCourseIDs :many
SELECT id, course_id, day_of_week, slot_number, synced_at
FROM course_sessions_cache
WHERE course_id = ANY($1::uuid[])
ORDER BY course_id, day_of_week, slot_number;

-- name: UpsertCourseSession :one
INSERT INTO course_sessions_cache (
    id, course_id, day_of_week, slot_number, synced_at
) VALUES (
    $1, $2, $3, $4, NOW()
)
ON CONFLICT (course_id, day_of_week, slot_number) DO UPDATE SET
    synced_at = NOW()
RETURNING id, course_id, day_of_week, slot_number, synced_at;

-- name: DeleteCourseSessionsByCourseID :exec
DELETE FROM course_sessions_cache
WHERE course_id = $1;

-- name: CheckScheduleConflict :many
SELECT cs1.course_id as course_id_1, cs2.course_id as course_id_2, cs1.day_of_week, cs1.slot_number
FROM course_sessions_cache cs1
JOIN course_sessions_cache cs2
  ON cs1.day_of_week = cs2.day_of_week
  AND cs1.slot_number = cs2.slot_number
WHERE cs1.course_id = ANY($1::uuid[])
  AND cs2.course_id = ANY($1::uuid[])
  AND cs1.course_id != cs2.course_id;

-- name: CheckScheduleConflictWithExisting :many
-- Checks if any new courses conflict with courses in the student's approved enrollment
SELECT cs1.course_id as course_id_1, cs2.course_id as course_id_2, cs1.day_of_week, cs1.slot_number
FROM course_sessions_cache cs1
JOIN course_sessions_cache cs2
  ON cs1.day_of_week = cs2.day_of_week
  AND cs1.slot_number = cs2.slot_number
JOIN enrollment_program_courses epc ON cs2.course_id = epc.course_id
JOIN enrollment_programs ep ON epc.program_id = ep.id
WHERE cs1.course_id = ANY($1::uuid[])
  AND ep.student_id = $2
  AND ep.status = 'approved'
  AND cs1.course_id != cs2.course_id;
