-- catalog database: the active semester, its course offerings, and the
-- separate Bahar enrollment window the prerequisite demo needs.
--
-- Instructor ids come from _staff — staff.staff is another database now.
\i /seed/sql/_staging-for-catalog.sql

-- ============================================================
-- Active semester
-- ============================================================
INSERT INTO course_catalog.semesters (name, status, hard_deadline, activated_at)
VALUES ('2025-2026 Güz', 'active', NOW() + INTERVAL '60 days', NOW())
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- Course offerings for the semester (each assigned to a teacher)
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
JOIN _staff s ON s.email = m.email
ON CONFLICT (semester, course_code, department) DO NOTHING;

-- ============================================================
-- Prerequisite demo — CENG201 (Bahar) requires CENG102
-- ============================================================
-- A standalone enrollment window in a SEPARATE semester ('2025-2026 Bahar')
-- so nobody has an existing approved program blocking a fresh submission (the
-- '2025-2026 Güz' programs are approved and per-semester exclusive). The
-- enrollment period check reads enrollment.academic_periods, a projection of
-- catalog's enrollment-typed row — no 'semesters' row is needed, so the
-- single-active-semester constraint is untouched.
--
-- Expected behaviour when a student POSTs a program for '2025-2026 Bahar'
-- containing the Bahar CENG201 offering:
--   accept  → Zeynep, Emir, Elif, Baran (passed CENG102, class_level ≥ 2)
--   reject  → Deniz, Selin  (class_level 2, but never passed CENG102 →
--             ErrPrerequisitesNotMet — the exact path this feature adds)
--   reject  → Kerem, Naz    (class_level 1 < 2 → ErrInvalidClassLevel, a
--             different, earlier guard — not the prerequisite check)

-- Catalog holds one row per consuming service (source of truth); the services
-- themselves read local projections, normally filled by catalog's period
-- events. The seed writes those projections directly in the per-service files,
-- same shortcut it already takes for the view tables.
INSERT INTO course_catalog.academic_periods (semester, period_start, period_end, is_active, period_type)
SELECT '2025-2026 Bahar', NOW() - INTERVAL '1 day', NOW() + INTERVAL '30 days', true, t.period_type
FROM (VALUES ('catalog'), ('enrollment'), ('grading'), ('attendance')) AS t(period_type)
ON CONFLICT (semester, period_type) DO NOTHING;

-- Bahar CENG201 offering, carrying the prerequisite snapshot the enrollment
-- check walks (semester_courses.prerequisites, keyed by course_code). The id in
-- the snapshot is CENG102's catalog id; the check matches on course_code only.
INSERT INTO course_catalog.semester_courses
  (semester, course_code, department, credits, class_level, instructor_id, instructor_fullname,
   classroom_location, max_capacity, prerequisites, assessment_schema)
SELECT '2025-2026 Bahar', c.course_code, c.department, c.credits, c.class_level, s.id,
       s.first_name || ' ' || s.last_name, 'A Blok 401', 40,
       jsonb_build_array(jsonb_build_object(
         'id', (SELECT id FROM course_catalog.course_catalog WHERE course_code = 'CENG102'),
         'course_code', 'CENG102',
         'course_name', 'Veri Yapıları')),
       '[{"slug":"midterm","name":"Vize","weight":40},{"slug":"final","name":"Final","weight":60}]'::jsonb
FROM course_catalog.course_catalog c
JOIN _staff s ON s.email = 'ayse.demir@uni.edu.tr'
WHERE c.course_code = 'CENG201'
ON CONFLICT (semester, course_code, department) DO NOTHING;
