-- +goose Up
CREATE TABLE IF NOT EXISTS staff (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'teacher',
    department VARCHAR(100),
    phone VARCHAR(20),
    office_location VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT true,
    deleted_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_staff_email ON staff(email);
CREATE INDEX idx_staff_is_active ON staff(is_active);
CREATE INDEX idx_staff_deleted_at ON staff(deleted_at);
CREATE INDEX idx_staff_department ON staff(department);

-- +goose Down
DROP TABLE IF EXISTS staff;
