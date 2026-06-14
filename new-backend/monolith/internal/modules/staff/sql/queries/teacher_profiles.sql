-- name: GetTeacherProfileByStaffID :one
SELECT tp.id, tp.staff_id, tp.academic_title, tp.faculty, tp.profile_image_url,
       tp.education, tp.articles, tp.bulletins, tp.projects, tp.awards,
       tp.scholarships, tp.admin_assignments, tp.created_at, tp.updated_at,
       s.email, s.first_name, s.last_name, s.department, s.phone, s.office_location
FROM staff.teacher_profiles tp
JOIN staff.staff s ON tp.staff_id = s.id
WHERE tp.staff_id = $1 AND s.is_active = true
LIMIT 1;

-- name: CreateTeacherProfile :one
INSERT INTO staff.teacher_profiles (staff_id, academic_title, faculty, profile_image_url)
VALUES ($1, $2, $3, $4)
RETURNING id, staff_id, academic_title, faculty, profile_image_url,
          education, articles, bulletins, projects, awards,
          scholarships, admin_assignments, created_at, updated_at;

-- name: UpdateTeacherProfile :one
UPDATE staff.teacher_profiles
SET academic_title = COALESCE($2, academic_title),
    faculty = COALESCE($3, faculty),
    profile_image_url = COALESCE($4, profile_image_url),
    education = COALESCE($5, education),
    articles = COALESCE($6, articles),
    bulletins = COALESCE($7, bulletins),
    projects = COALESCE($8, projects),
    awards = COALESCE($9, awards),
    scholarships = COALESCE($10, scholarships),
    admin_assignments = COALESCE($11, admin_assignments),
    updated_at = NOW()
WHERE staff_id = $1
RETURNING id, staff_id, academic_title, faculty, profile_image_url,
          education, articles, bulletins, projects, awards,
          scholarships, admin_assignments, created_at, updated_at;

-- name: DeleteTeacherProfileByStaffID :exec
DELETE FROM staff.teacher_profiles WHERE staff_id = $1;

-- name: ListTeacherProfiles :many
SELECT tp.id, tp.staff_id, tp.academic_title, tp.faculty, tp.profile_image_url,
       tp.education, tp.articles, tp.bulletins, tp.projects, tp.awards,
       tp.scholarships, tp.admin_assignments, tp.created_at, tp.updated_at,
       s.email, s.first_name, s.last_name, s.department, s.phone, s.office_location
FROM staff.teacher_profiles tp
JOIN staff.staff s ON tp.staff_id = s.id
WHERE s.is_active = true
ORDER BY s.last_name, s.first_name
LIMIT $1 OFFSET $2;

-- name: CountTeacherProfiles :one
SELECT COUNT(*) FROM staff.teacher_profiles tp
JOIN staff.staff s ON tp.staff_id = s.id
WHERE s.is_active = true;
