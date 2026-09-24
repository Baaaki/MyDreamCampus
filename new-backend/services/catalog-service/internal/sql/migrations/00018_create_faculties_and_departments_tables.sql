-- +goose Up
-- Reference data for the faculty/department pickers. Courses, students and
-- staff keep their faculty/department as free text: those rows live in other
-- databases, so a foreign key could not reach them anyway — an exact name
-- match is the contract.
--
-- slug is the public identifier: the frontend puts it in URLs and filters
-- (`fac-egitim`, `dept-bote`), so it must stay stable across re-seeds while
-- the uuid does not.
CREATE TABLE course_catalog.faculties (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    slug VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL,
    -- Same width as course_catalog.faculty, whose values must match it.
    name VARCHAR(100) NOT NULL UNIQUE,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE course_catalog.departments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    faculty_id UUID NOT NULL REFERENCES course_catalog.faculties(id) ON DELETE CASCADE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL,
    -- Unique per faculty only: two faculties can both teach "İktisat".
    name VARCHAR(100) NOT NULL,
    description TEXT,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (faculty_id, name)
);

CREATE INDEX idx_departments_faculty_id ON course_catalog.departments(faculty_id);

-- +goose Down
DROP TABLE IF EXISTS course_catalog.departments;
DROP TABLE IF EXISTS course_catalog.faculties;
