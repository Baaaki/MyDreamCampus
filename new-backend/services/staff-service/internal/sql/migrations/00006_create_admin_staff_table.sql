-- +goose Up
-- Administrative staff (faculty secretaries, student-affairs officers) are a
-- directory admins maintain, not accounts. Keeping them out of staff.staff
-- is what keeps them out of staff.created — the event auth turns into a
-- login, and whose roles (teacher/admin/student) have no place for them.
CREATE TABLE IF NOT EXISTS staff.admin_staff (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    email VARCHAR(255) NOT NULL UNIQUE,
    title VARCHAR(50) NOT NULL DEFAULT '',
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    faculty VARCHAR(200) NOT NULL,
    department VARCHAR(100) NOT NULL DEFAULT '',
    phone VARCHAR(20) NOT NULL DEFAULT '',
    profile_image_url TEXT NOT NULL DEFAULT '',
    position VARCHAR(150) NOT NULL,
    job_description TEXT NOT NULL DEFAULT '',
    responsibilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    working_hours VARCHAR(100) NOT NULL DEFAULT '',
    office_location VARCHAR(200) NOT NULL DEFAULT '',
    start_date DATE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_staff_faculty ON staff.admin_staff(faculty) WHERE is_active;

-- +goose Down
DROP TABLE IF EXISTS staff.admin_staff;
