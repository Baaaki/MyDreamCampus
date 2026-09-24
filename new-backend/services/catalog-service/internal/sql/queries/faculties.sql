-- name: ListFaculties :many
SELECT * FROM course_catalog.faculties
ORDER BY sort_order, name;

-- name: ListDepartments :many
SELECT * FROM course_catalog.departments
ORDER BY faculty_id, sort_order, name;
