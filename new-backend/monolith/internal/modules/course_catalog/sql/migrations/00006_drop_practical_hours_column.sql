-- +goose Up
ALTER TABLE course_catalog.course_catalog DROP COLUMN practical_hours;

-- +goose Down
ALTER TABLE course_catalog.course_catalog ADD COLUMN practical_hours SMALLINT NOT NULL DEFAULT 0 CHECK (practical_hours >= 0 AND practical_hours <= 20);
