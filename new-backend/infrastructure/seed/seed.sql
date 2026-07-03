-- Relational demo data, layered on top of the users/courses the API seed
-- created. Runs after those exist. Everything is idempotent (ON CONFLICT
-- DO NOTHING) so a re-run is safe and it won't fight event-driven projections
-- that may have already filled a view table.
--
-- No cross-schema FKs exist in this system (modules link by logical UUID), so
-- each module's tables — including grades' read-model views — are filled here
-- directly and stay self-consistent.

-- ============================================================
-- 1. Teacher profiles (rich detail: title, education, awards, ...)
-- ============================================================
INSERT INTO staff.teacher_profiles (staff_id, academic_title, faculty, education, awards, articles, projects)
SELECT s.id, m.title, 'Mühendislik Fakültesi',
       m.education::jsonb, m.awards::jsonb, m.articles::jsonb, m.projects::jsonb
FROM staff.staff s
JOIN (VALUES
  ('ahmet.yilmaz@uni.edu.tr', 'Prof. Dr.',
   '[{"degree":"Lisans","school":"Boğaziçi Üniversitesi","field":"Bilgisayar Mühendisliği","year":2001},{"degree":"Doktora","school":"ODTÜ","field":"Bilgisayar Mühendisliği","year":2009}]',
   '[{"title":"TÜBİTAK Bilim Ödülü","year":2018},{"title":"Yılın Öğretim Üyesi","year":2021}]',
   '[{"title":"Deep Learning for Graph Networks","journal":"IEEE Access","year":2020}]',
   '[{"name":"Akıllı Kampüs Platformu","role":"Yürütücü","year":2022}]'),
  ('ayse.demir@uni.edu.tr', 'Doç. Dr.',
   '[{"degree":"Lisans","school":"İTÜ","field":"Bilgisayar Mühendisliği","year":2006},{"degree":"Doktora","school":"Sabancı Üniversitesi","field":"Yapay Zeka","year":2014}]',
   '[{"title":"En İyi Bildiri Ödülü, UBMK","year":2019}]',
   '[{"title":"Explainable AI in Education","journal":"Springer LNCS","year":2021}]',
   '[{"name":"Öğrenci Başarı Tahmini","role":"Araştırmacı","year":2023}]'),
  ('mehmet.kaya@uni.edu.tr', 'Dr. Öğr. Üyesi',
   '[{"degree":"Lisans","school":"Hacettepe Üniversitesi","field":"Elektrik-Elektronik Mühendisliği","year":2010},{"degree":"Doktora","school":"Bilkent Üniversitesi","field":"Gömülü Sistemler","year":2018}]',
   '[{"title":"Genç Bilim İnsanı Teşvik Ödülü","year":2020}]',
   '[{"title":"Low-Power IoT Scheduling","journal":"Elsevier IoT","year":2022}]',
   '[{"name":"Kampüs Enerji İzleme","role":"Yürütücü","year":2024}]')
 ) AS m(email, title, education, awards, articles, projects) ON m.email = s.email
ON CONFLICT (staff_id) DO NOTHING;

-- ============================================================
-- 2. Active semester
-- ============================================================
INSERT INTO course_catalog.semesters (name, status, hard_deadline, activated_at)
VALUES ('2025-2026 Güz', 'active', NOW() + INTERVAL '60 days', NOW())
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- 3. Course offerings for the semester (each assigned to a teacher)
-- ============================================================
INSERT INTO course_catalog.semester_courses
  (semester, course_code, department, credits, class_level, instructor_id, instructor_fullname,
   classroom_location, max_capacity, assessment_schema)
SELECT '2025-2026 Güz', c.course_code, c.department, c.credits, c.class_level, s.id,
       s.first_name || ' ' || s.last_name,
       'A Blok ' || (300 + (row_number() OVER (ORDER BY c.course_code)))::text,
       40,
       '[{"slug":"midterm","name":"Vize","weight":40},{"slug":"final","name":"Final","weight":60}]'::jsonb
FROM course_catalog.course_catalog c
JOIN (VALUES
   ('CENG101','ahmet.yilmaz@uni.edu.tr'),
   ('CENG102','ahmet.yilmaz@uni.edu.tr'),
   ('CENG201','ayse.demir@uni.edu.tr'),
   ('CENG202','ayse.demir@uni.edu.tr'),
   ('CENG301','mehmet.kaya@uni.edu.tr'),
   ('CENG350','mehmet.kaya@uni.edu.tr')
 ) AS m(course_code, email) ON m.course_code = c.course_code
JOIN staff.staff s ON s.email = m.email
ON CONFLICT (semester, course_code, department) DO NOTHING;

-- ============================================================
-- 4. grades read-model views (normally filled by events; fill directly)
-- ============================================================
INSERT INTO grades.students_view (id, student_number, first_name, last_name, email, department, class_level, is_active)
SELECT id, student_number, first_name, last_name, email, department, class_level, is_active
FROM student.students
ON CONFLICT (id) DO NOTHING;

INSERT INTO grades.courses_view
  (id, course_code, course_name, credits, semester, department, instructor_id, instructor_fullname, assessment_schema)
SELECT sc.id, sc.course_code, c.name, sc.credits, sc.semester, c.department,
       sc.instructor_id, sc.instructor_fullname, sc.assessment_schema
FROM course_catalog.semester_courses sc
JOIN course_catalog.course_catalog c ON c.course_code = sc.course_code
WHERE sc.semester = '2025-2026 Güz'
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 5. Enrollment programs (approved) — every student takes 3 courses
-- ============================================================
INSERT INTO enrollment.enrollment_programs (student_id, semester, status)
SELECT id, '2025-2026 Güz', 'approved' FROM student.students
ON CONFLICT (student_id, semester) DO NOTHING;

INSERT INTO enrollment.enrollment_program_courses (program_id, course_id, course_code, course_name, credits)
SELECT p.id, c.id, c.course_code, c.name, c.credits
FROM enrollment.enrollment_programs p
JOIN course_catalog.course_catalog c ON c.course_code IN ('CENG101','CENG102','CENG201')
WHERE p.semester = '2025-2026 Güz'
ON CONFLICT (program_id, course_id) DO NOTHING;

-- ============================================================
-- 6. Grade registrations + midterm/final scores for those courses
-- ============================================================
INSERT INTO grades.student_course_registrations (student_id, course_id, semester)
SELECT sv.id, cv.id, '2025-2026 Güz'
FROM grades.students_view sv
JOIN grades.courses_view cv
  ON cv.semester = '2025-2026 Güz' AND cv.course_code IN ('CENG101','CENG102','CENG201')
ON CONFLICT (student_id, course_id) DO NOTHING;

INSERT INTO grades.student_assessment_scores (registration_id, slug, score, graded_by)
SELECT r.id, v.slug, v.score, cv.instructor_id
FROM grades.student_course_registrations r
JOIN grades.courses_view cv ON cv.id = r.course_id
CROSS JOIN LATERAL (VALUES
   ('midterm', (55 + floor(random() * 40))::numeric(5,2)),
   ('final',   (50 + floor(random() * 45))::numeric(5,2))
 ) AS v(slug, score)
ON CONFLICT (registration_id, slug) DO NOTHING;
