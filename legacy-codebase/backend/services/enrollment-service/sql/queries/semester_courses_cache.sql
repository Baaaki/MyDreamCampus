-- name: GetAvailableCourses :many
SELECT id, course_code, course_name, faculty, department, credits, course_type, class_level,
       semester, instructor_id, instructor_fullname, classroom_location,
       max_capacity, current_enrollment, prerequisites, synced_at
FROM semester_courses_cache
WHERE department = $1
  AND class_level <= $2
  AND semester = $3
ORDER BY class_level, course_code;

-- name: GetCourseByID :one
SELECT id, course_code, course_name, faculty, department, credits, course_type, class_level,
       semester, instructor_id, instructor_fullname, classroom_location,
       max_capacity, current_enrollment, prerequisites, synced_at
FROM semester_courses_cache
WHERE id = $1
LIMIT 1;

-- name: GetCoursesByIDs :many
SELECT id, course_code, course_name, faculty, department, credits, course_type, class_level,
       semester, instructor_id, instructor_fullname, classroom_location,
       max_capacity, current_enrollment, prerequisites, synced_at
FROM semester_courses_cache
WHERE id = ANY($1::uuid[])
ORDER BY id;

-- name: UpsertSemesterCourse :one
INSERT INTO semester_courses_cache (
    id, course_code, course_name, faculty, department, credits, course_type, class_level,
    semester, instructor_id, instructor_fullname, classroom_location, max_capacity,
    current_enrollment, prerequisites, synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW()
)
ON CONFLICT (id) DO UPDATE SET
    course_code = EXCLUDED.course_code,
    course_name = EXCLUDED.course_name,
    faculty = EXCLUDED.faculty,
    department = EXCLUDED.department,
    credits = EXCLUDED.credits,
    course_type = EXCLUDED.course_type,
    class_level = EXCLUDED.class_level,
    semester = EXCLUDED.semester,
    instructor_id = EXCLUDED.instructor_id,
    instructor_fullname = EXCLUDED.instructor_fullname,
    classroom_location = EXCLUDED.classroom_location,
    max_capacity = EXCLUDED.max_capacity,
    prerequisites = EXCLUDED.prerequisites,
    synced_at = NOW()
RETURNING id, course_code, course_name, faculty, department, credits, course_type, class_level,
          semester, instructor_id, instructor_fullname, classroom_location,
          max_capacity, current_enrollment, prerequisites, synced_at;

-- name: IncrementEnrollment :execrows
UPDATE semester_courses_cache
SET current_enrollment = current_enrollment + 1
WHERE id = $1 AND current_enrollment < max_capacity;

-- name: DecrementEnrollment :execrows
UPDATE semester_courses_cache
SET current_enrollment = current_enrollment - 1
WHERE id = $1 AND current_enrollment > 0;

-- name: GetCoursesForCapacityCheck :many
SELECT id, current_enrollment, max_capacity
FROM semester_courses_cache
WHERE id = ANY($1::uuid[])
ORDER BY id
FOR UPDATE;
