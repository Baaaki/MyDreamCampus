-- +goose Up
-- Trigger function to prevent instructor schedule conflicts
-- This ensures the same instructor cannot be assigned to multiple courses at the same time slot

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_instructor_schedule_conflict()
RETURNS TRIGGER AS $$
DECLARE
    v_instructor_id UUID;
    v_semester VARCHAR(50);
    v_conflict_course_code VARCHAR(50);
BEGIN
    -- Get instructor_id and semester from course_catalog.semester_courses
    SELECT instructor_id, semester
    INTO v_instructor_id, v_semester
    FROM course_catalog.semester_courses
    WHERE id = NEW.semester_course_id;

    -- Check if instructor has another course at the same time slot
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

    -- If conflict exists, raise exception
    IF v_conflict_course_code IS NOT NULL THEN
        RAISE EXCEPTION 'Instructor schedule conflict: already teaching % at this time slot', v_conflict_course_code
            USING ERRCODE = '23505'; -- unique_violation error code
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- Create trigger that runs before INSERT or UPDATE
CREATE TRIGGER trg_instructor_schedule_conflict
    BEFORE INSERT OR UPDATE ON course_catalog.course_schedule_sessions
    FOR EACH ROW
    EXECUTE FUNCTION check_instructor_schedule_conflict();

COMMENT ON FUNCTION check_instructor_schedule_conflict() IS
'Prevents instructor from being assigned to multiple courses at the same time slot in the same semester';

-- +goose Down
DROP TRIGGER IF EXISTS trg_instructor_schedule_conflict ON course_catalog.course_schedule_sessions;
DROP FUNCTION IF EXISTS check_instructor_schedule_conflict();
