-- name: GetCourseByCourseCode :one
SELECT id, course_code, name, faculty, department, offering_unit,
       class_level, semester, credits, ects, theoretical_hours, lab_hours,
       course_type, course_category, education_level, teaching_type, language,
       prerequisites, coordinator, purpose, description, learning_outcomes,
       learning_outcomes_list, weekly_topics, recommended_sources, syllabus,
       status, created_at, updated_at
FROM course_catalog.course_catalog
WHERE course_code = $1
LIMIT 1;

-- name: CreateCourse :one
INSERT INTO course_catalog.course_catalog (
    course_code, name, faculty, department, offering_unit,
    class_level, semester, credits, ects, theoretical_hours, lab_hours,
    course_type, course_category, education_level, teaching_type, language,
    prerequisites, coordinator, purpose, description, learning_outcomes,
    learning_outcomes_list, weekly_topics, recommended_sources, syllabus, status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
RETURNING id, course_code, name, faculty, department, offering_unit,
          class_level, semester, credits, ects, theoretical_hours, lab_hours,
          course_type, course_category, education_level, teaching_type, language,
          prerequisites, coordinator, purpose, description, learning_outcomes,
          learning_outcomes_list, weekly_topics, recommended_sources, syllabus,
          status, created_at, updated_at;

-- name: UpdateCourse :one
UPDATE course_catalog.course_catalog
SET name = COALESCE(sqlc.narg('name'), name),
    faculty = COALESCE(sqlc.narg('faculty'), faculty),
    department = COALESCE(sqlc.narg('department'), department),
    offering_unit = COALESCE(sqlc.narg('offering_unit'), offering_unit),
    class_level = COALESCE(sqlc.narg('class_level'), class_level),
    semester = COALESCE(sqlc.narg('semester'), semester),
    credits = COALESCE(sqlc.narg('credits'), credits),
    ects = COALESCE(sqlc.narg('ects'), ects),
    theoretical_hours = COALESCE(sqlc.narg('theoretical_hours'), theoretical_hours),
    lab_hours = COALESCE(sqlc.narg('lab_hours'), lab_hours),
    course_type = COALESCE(sqlc.narg('course_type'), course_type),
    course_category = COALESCE(sqlc.narg('course_category'), course_category),
    education_level = COALESCE(sqlc.narg('education_level'), education_level),
    teaching_type = COALESCE(sqlc.narg('teaching_type'), teaching_type),
    language = COALESCE(sqlc.narg('language'), language),
    prerequisites = COALESCE(sqlc.narg('prerequisites'), prerequisites),
    coordinator = COALESCE(sqlc.narg('coordinator'), coordinator),
    purpose = COALESCE(sqlc.narg('purpose'), purpose),
    description = COALESCE(sqlc.narg('description'), description),
    learning_outcomes = COALESCE(sqlc.narg('learning_outcomes'), learning_outcomes),
    learning_outcomes_list = COALESCE(sqlc.narg('learning_outcomes_list'), learning_outcomes_list),
    weekly_topics = COALESCE(sqlc.narg('weekly_topics'), weekly_topics),
    recommended_sources = COALESCE(sqlc.narg('recommended_sources'), recommended_sources),
    syllabus = COALESCE(sqlc.narg('syllabus'), syllabus),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = NOW()
WHERE course_code = $1
RETURNING id, course_code, name, faculty, department, offering_unit,
          class_level, semester, credits, ects, theoretical_hours, lab_hours,
          course_type, course_category, education_level, teaching_type, language,
          prerequisites, coordinator, purpose, description, learning_outcomes,
          learning_outcomes_list, weekly_topics, recommended_sources, syllabus,
          status, created_at, updated_at;

-- name: ListCourses :many
SELECT id, course_code, name, faculty, department, offering_unit,
       class_level, semester, credits, ects, theoretical_hours, lab_hours,
       course_type, course_category, education_level, teaching_type, language,
       prerequisites, status
FROM course_catalog.course_catalog
WHERE (sqlc.narg('faculty')::text IS NULL OR faculty = sqlc.narg('faculty'))
  AND (sqlc.narg('department')::text IS NULL OR department = sqlc.narg('department'))
  AND (sqlc.narg('course_type')::course_catalog.course_type_enum IS NULL OR course_type = sqlc.narg('course_type'))
  AND (sqlc.narg('course_category')::course_catalog.course_category_enum IS NULL OR course_category = sqlc.narg('course_category'))
  AND (sqlc.narg('education_level')::course_catalog.education_level_enum IS NULL OR education_level = sqlc.narg('education_level'))
  AND (sqlc.narg('status')::course_catalog.course_catalog_status_enum IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('class_level')::SMALLINT IS NULL OR class_level = sqlc.narg('class_level'))
  AND (sqlc.narg('semester')::SMALLINT IS NULL OR semester = sqlc.narg('semester'))
  AND (sqlc.narg('language')::text IS NULL OR language = sqlc.narg('language'))
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%' OR course_code ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY course_code
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountCourses :one
SELECT COUNT(*)
FROM course_catalog.course_catalog
WHERE (sqlc.narg('faculty')::text IS NULL OR faculty = sqlc.narg('faculty'))
  AND (sqlc.narg('department')::text IS NULL OR department = sqlc.narg('department'))
  AND (sqlc.narg('course_type')::course_catalog.course_type_enum IS NULL OR course_type = sqlc.narg('course_type'))
  AND (sqlc.narg('course_category')::course_catalog.course_category_enum IS NULL OR course_category = sqlc.narg('course_category'))
  AND (sqlc.narg('education_level')::course_catalog.education_level_enum IS NULL OR education_level = sqlc.narg('education_level'))
  AND (sqlc.narg('status')::course_catalog.course_catalog_status_enum IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('class_level')::SMALLINT IS NULL OR class_level = sqlc.narg('class_level'))
  AND (sqlc.narg('semester')::SMALLINT IS NULL OR semester = sqlc.narg('semester'))
  AND (sqlc.narg('language')::text IS NULL OR language = sqlc.narg('language'))
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%' OR course_code ILIKE '%' || sqlc.narg('search') || '%');

-- name: GetCourseByID :one
SELECT id, course_code, name, faculty, department, offering_unit,
       class_level, semester, credits, ects, theoretical_hours, lab_hours,
       course_type, course_category, education_level, teaching_type, language,
       prerequisites, coordinator, purpose, description, learning_outcomes,
       learning_outcomes_list, weekly_topics, recommended_sources, syllabus,
       status, created_at, updated_at
FROM course_catalog.course_catalog
WHERE id = $1
LIMIT 1;

-- name: GetCoursesByIDs :many
SELECT id, course_code, name, class_level
FROM course_catalog.course_catalog
WHERE id = ANY($1::uuid[]);
