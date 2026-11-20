-- +goose Up
-- Serialize concurrent schedule inserts per (semester, instructor_id) with a
-- transaction-scoped advisory lock so two overlapping transactions under READ
-- COMMITTED cannot both pass the conflict SELECT and double-book an instructor.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_instructor_schedule_conflict()
RETURNS TRIGGER AS $$
DECLARE
    v_instructor_id UUID;
    v_semester VARCHAR(50);
    v_conflict_course_code VARCHAR(50);
BEGIN
    SELECT instructor_id, semester
    INTO v_instructor_id, v_semester
    FROM course_catalog.semester_courses
    WHERE id = NEW.semester_course_id;

    PERFORM pg_advisory_xact_lock(hashtext(v_semester || ':' || v_instructor_id::text));

    SELECT sc.course_code
    INTO v_conflict_course_code
    FROM course_catalog.course_schedule_sessions css
    JOIN course_catalog.semester_courses sc ON css.semester_course_id = sc.id
    WHERE css.id != COALESCE(NEW.id, '00000000-0000-0000-0000-000000000000'::uuid)
      AND sc.instructor_id = v_instructor_id
      AND sc.semester = v_semester
      AND css.day_of_week = NEW.day_of_week
      AND css.slot_number = NEW.slot_number
    LIMIT 1;

    IF v_conflict_course_code IS NOT NULL THEN
        RAISE EXCEPTION 'Instructor schedule conflict: already teaching % at this time slot', v_conflict_course_code
            USING ERRCODE = '23505';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_instructor_schedule_conflict()
RETURNS TRIGGER AS $$
DECLARE
    v_instructor_id UUID;
    v_semester VARCHAR(50);
    v_conflict_course_code VARCHAR(50);
BEGIN
    SELECT instructor_id, semester
    INTO v_instructor_id, v_semester
    FROM course_catalog.semester_courses
    WHERE id = NEW.semester_course_id;

    SELECT sc.course_code
    INTO v_conflict_course_code
    FROM course_catalog.course_schedule_sessions css
    JOIN course_catalog.semester_courses sc ON css.semester_course_id = sc.id
    WHERE css.id != COALESCE(NEW.id, '00000000-0000-0000-0000-000000000000'::uuid)
      AND sc.instructor_id = v_instructor_id
      AND sc.semester = v_semester
      AND css.day_of_week = NEW.day_of_week
      AND css.slot_number = NEW.slot_number
    LIMIT 1;

    IF v_conflict_course_code IS NOT NULL THEN
        RAISE EXCEPTION 'Instructor schedule conflict: already teaching % at this time slot', v_conflict_course_code
            USING ERRCODE = '23505';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
