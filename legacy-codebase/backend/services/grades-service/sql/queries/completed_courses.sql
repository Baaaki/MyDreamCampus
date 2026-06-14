-- name: CreateCompletedCourse :one
INSERT INTO student_completed_courses (
    student_id, student_number, student_first_name, student_last_name, student_department,
    course_id, course_code, course_name, credits, semester,
    instructor_id, instructor_name,
    assessment_scores, weighted_average, grade_point,
    grading_type, grading_config, class_statistics,
    is_attendance_failed, finalized_at, finalized_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12,
    $13, $14, $15,
    $16, $17, $18,
    $19, $20, $21
)
RETURNING *;

-- name: DeleteCompletedCourse :exec
DELETE FROM student_completed_courses
WHERE student_id = $1 AND course_code = $2;

-- name: GetCompletedCoursesByStudent :many
SELECT * FROM student_completed_courses
WHERE student_id = $1
ORDER BY finalized_at DESC;

-- name: GetCompletedCoursesByCourse :many
SELECT * FROM student_completed_courses
WHERE course_id = $1
ORDER BY student_number;

-- name: CalculateStudentGPA :one
SELECT
    COALESCE(
        ROUND(
            SUM(grade_point::text::decimal * credits) / NULLIF(SUM(credits), 0),
            2
        ), 0
    ) as gpa,
    COALESCE(SUM(credits), 0) as total_credits
FROM student_completed_courses
WHERE student_id = $1;

-- name: GetTranscriptData :many
SELECT
    semester,
    course_code,
    course_name,
    credits,
    grade_point,
    finalized_at
FROM student_completed_courses
WHERE student_id = $1
ORDER BY finalized_at;

-- name: GetCompletedCourseByStudentAndCourse :one
SELECT * FROM student_completed_courses
WHERE student_id = $1 AND course_id = $2;

-- name: UpdateCompletedCourseAfterAppeal :exec
UPDATE student_completed_courses
SET
    assessment_scores = $3,
    weighted_average = $4,
    grade_point = $5,
    grading_config = $6,
    finalized_at = NOW()
WHERE student_id = $1 AND course_id = $2;
