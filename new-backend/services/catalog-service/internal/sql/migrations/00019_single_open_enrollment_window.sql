-- +goose Up
-- Only one semester can take enrollments at a time: an active enrollment
-- window may not overlap another semester's active one. A trigger rather than
-- an EXCLUDE constraint so the rows already there are left alone — an
-- installation seeded with two overlapping windows still migrates, and the
-- admin closes one of them from the UI (deactivating is always allowed).
--
-- Every write path (period CRUD, the semester wizard, the seed) goes through
-- this table, so the rule holds without each of them checking on its own.
-- The advisory lock serialises concurrent writers: two admins opening two
-- semesters at once would otherwise each see the other's row as not yet there.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION course_catalog.check_single_enrollment_window()
RETURNS TRIGGER AS $$
DECLARE
    v_conflict_semester VARCHAR(50);
BEGIN
    IF NEW.period_type <> 'enrollment' OR NOT NEW.is_active THEN
        RETURN NEW;
    END IF;

    PERFORM pg_advisory_xact_lock(hashtext('course_catalog.enrollment_window'));

    SELECT p.semester
    INTO v_conflict_semester
    FROM course_catalog.academic_periods p
    WHERE p.period_type = 'enrollment'
      AND p.is_active
      AND p.semester <> NEW.semester
      AND p.period_start <= NEW.period_end
      AND p.period_end >= NEW.period_start
    LIMIT 1;

    IF v_conflict_semester IS NOT NULL THEN
        RAISE EXCEPTION 'enrollment window overlaps the open window of semester %', v_conflict_semester
            USING ERRCODE = 'exclusion_violation';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_single_enrollment_window
    BEFORE INSERT OR UPDATE ON course_catalog.academic_periods
    FOR EACH ROW
    EXECUTE FUNCTION course_catalog.check_single_enrollment_window();

-- +goose Down
DROP TRIGGER IF EXISTS trg_single_enrollment_window ON course_catalog.academic_periods;
DROP FUNCTION IF EXISTS course_catalog.check_single_enrollment_window();
