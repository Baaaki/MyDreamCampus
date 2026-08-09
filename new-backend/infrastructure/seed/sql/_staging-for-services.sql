-- Staging rows the grades / attendance / enrollment / meal seeds need from
-- OTHER databases: students from the student database, courses, semester
-- offerings and academic periods from catalog. Exported by seed.sh after the
-- catalog seed has run, so the offerings and the Bahar periods already exist.
--
-- All four are loaded in every one of those sessions even where only some are
-- read — a few hundred rows, and one file beats four near-identical ones.
CREATE TEMP TABLE _students (
    id             uuid,
    student_number text,
    first_name     text,
    last_name      text,
    email          text,
    department     text,
    class_level    smallint,
    is_active      boolean
);
\copy _students FROM '/tmp/seed-stage/students.csv' CSV

CREATE TEMP TABLE _courses (
    id          uuid,
    course_code text,
    name        text,
    credits     smallint,
    department  text,
    class_level smallint
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
    assessment_schema   jsonb
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
