-- Staging rows the grades / attendance / enrollment / meal seeds need from
-- OTHER databases: students from the student database, courses, semester
-- offerings and academic periods from catalog. Exported by seed.sh after the
-- catalog seed has run, so the offerings and periods already exist.
--
-- All four are loaded in every one of those sessions even where only some are
-- read — a few hundred rows, and one file beats four near-identical ones.
CREATE TEMP TABLE _students (
    id              uuid,
    student_number  text,
    first_name      text,
    last_name       text,
    email           text,
    department      text,
    class_level     smallint,
    is_active       boolean,
    enrollment_year smallint,
    status          text
);
\copy _students FROM '/tmp/seed-stage/students.csv' CSV

CREATE TEMP TABLE _courses (
    id          uuid,
    course_code text,
    name        text,
    credits     smallint,
    department  text,
    class_level smallint,
    lab_hours   smallint
);
\copy _courses FROM '/tmp/seed-stage/courses.csv' CSV

CREATE TEMP TABLE _semester_courses (
    id                  uuid,
    semester            text,
    course_code         text,
    credits             smallint,
    class_level         smallint,
    instructor_id       uuid,
    instructor_fullname text,
    assessment_schema   jsonb,
    prerequisites       jsonb
);
\copy _semester_courses FROM '/tmp/seed-stage/semester_courses.csv' CSV

CREATE TEMP TABLE _periods (
    id           uuid,
    semester     text,
    period_start timestamptz,
    period_end   timestamptz,
    is_active    boolean,
    period_type  text
);
\copy _periods FROM '/tmp/seed-stage/periods.csv' CSV

-- ============================================================
-- The demo cohort, decided once for every service
-- ============================================================
-- Offerings exist for Bilgisayar Mühendisliği only (02-catalog.sql), so the
-- cohort is its active students. Each class year takes its own year's pair;
-- the fourth year shares the third year's, the catalog has no 4xx course.
-- Deniz and Selin never passed CENG102 (the prerequisite scenario below), so
-- they retake it instead of starting CENG201.
--
-- Three of ahmet.yilmaz's advisees have submitted programs he has not
-- approved yet — the advisor's inbox. A pending program has no grades or
-- attendance rows: those services only learn of it on approval.
CREATE TEMP TABLE _programs AS
SELECT st.id AS student_id, st.student_number, sc.id AS course_id, sc.course_code,
       c.name AS course_name, sc.credits,
       CASE WHEN st.student_number IN ('20210101001', '20210101002', '20220101025')
            THEN 'pending' ELSE 'approved' END AS status
FROM _students st
JOIN _semester_courses sc ON sc.semester = :'semester'
JOIN _courses c ON c.course_code = sc.course_code
WHERE st.department = 'Bilgisayar Mühendisliği'
  AND st.status = 'active'
  AND sc.course_code = ANY (CASE
        WHEN st.student_number IN ('2023510020', '2023510021') THEN ARRAY['CENG102', 'CENG202']
        WHEN st.class_level = 1 THEN ARRAY['CENG101', 'CENG102']
        WHEN st.class_level = 2 THEN ARRAY['CENG201', 'CENG202']
        ELSE ARRAY['CENG301', 'CENG350']
      END);

-- ============================================================
-- Transcript: what the cohort finished in earlier years
-- ============================================================
-- Semester names come from the enrollment year, in the same format as the
-- live ones. Scores are fixed for zeynep.sahin (taken from the old mock
-- transcript) and a stable hash for everyone else, so a re-seed reproduces
-- the same report cards. Every score lands at 60+, a pass under absolute
-- grading.
CREATE TEMP TABLE _completed AS
WITH taken AS (
    SELECT st.*, t.course_code, y.term_year, t.term
    FROM _students st
    CROSS JOIN (VALUES
        ('CENG101', 0, 'Fall',   2),
        ('CENG102', 0, 'Spring', 2),
        ('CENG201', 1, 'Fall',   3),
        ('CENG202', 1, 'Spring', 3)
    ) AS t(course_code, year_offset, term, min_class)
    CROSS JOIN LATERAL (SELECT st.enrollment_year + t.year_offset AS term_year) y
    WHERE st.department = 'Bilgisayar Mühendisliği'
      AND st.class_level >= t.min_class
      AND NOT (t.course_code = 'CENG102' AND st.student_number IN ('2023510020', '2023510021'))
), scored AS (
    SELECT tk.*,
           COALESCE(z.midterm, 60 + (abs(hashtext(tk.id::text || tk.course_code || 'm')) % 38))::numeric AS midterm,
           COALESCE(z.final,   60 + (abs(hashtext(tk.id::text || tk.course_code || 'f')) % 38))::numeric AS final
    FROM taken tk
    LEFT JOIN (VALUES
        ('CENG101', 88, 95), ('CENG102', 80, 88), ('CENG201', 85, 90), ('CENG202', 72, 82)
    ) AS z(course_code, midterm, final)
      ON tk.student_number = '2021510001' AND z.course_code = tk.course_code
)
SELECT s.id AS student_id, s.student_number, s.first_name, s.last_name, s.department,
       c.id AS course_id, s.course_code, c.name AS course_name, c.credits,
       format('%s-%s-%s', s.term_year, s.term_year + 1, s.term) AS semester,
       -- Fall ends in January, Spring in June.
       CASE s.term WHEN 'Fall' THEN make_date(s.term_year + 1, 1, 20)
                   ELSE make_date(s.term_year + 1, 6, 20) END AS finalized_on,
       s.midterm, s.final,
       round(0.4 * s.midterm + 0.6 * s.final, 2) AS weighted_average
FROM scored s
JOIN _courses c ON c.course_code = s.course_code;

-- grades' absolute scale (grading_helpers.go calculateAbsoluteGradePoint).
CREATE TEMP TABLE _grade_scale (min_average numeric, grade_point text);
INSERT INTO _grade_scale VALUES
    (90, '4.00'), (87.5, '3.75'), (85, '3.50'), (82.5, '3.25'), (80, '3.00'),
    (77.5, '2.75'), (75, '2.50'), (72.5, '2.25'), (70, '2.00'), (67.5, '1.75'),
    (65, '1.50'), (62.5, '1.25'), (60, '1.00'), (50, '0.50'), (0, '0.00');
