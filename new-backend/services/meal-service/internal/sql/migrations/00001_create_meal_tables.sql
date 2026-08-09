-- +goose Up
CREATE TABLE IF NOT EXISTS meal.cafeterias (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL,
    has_vegan_menu BOOLEAN NOT NULL DEFAULT false,
    serves_dinner BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cafeterias_is_active ON meal.cafeterias(is_active) WHERE is_active = true;

CREATE TABLE IF NOT EXISTS meal.students_view (
    id UUID PRIMARY KEY,
    student_number VARCHAR(50) NOT NULL UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_students_view_student_number ON meal.students_view(student_number);
CREATE INDEX idx_students_view_is_active ON meal.students_view(is_active) WHERE is_active = true;

CREATE TYPE meal.meal_time_enum AS ENUM ('lunch', 'dinner');
CREATE TYPE meal.menu_type_enum AS ENUM ('normal', 'vegan');
CREATE TYPE meal.reservation_status_enum AS ENUM ('pending', 'confirmed', 'cancelled', 'expired');

CREATE TABLE IF NOT EXISTS meal.reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NULL,
    student_id UUID NOT NULL REFERENCES meal.students_view(id) ON DELETE CASCADE,
    cafeteria_id UUID NOT NULL REFERENCES meal.cafeterias(id),
    reservation_date DATE NOT NULL,
    meal_time meal.meal_time_enum NOT NULL,
    menu_type meal.menu_type_enum NOT NULL DEFAULT 'normal',
    status meal.reservation_status_enum NOT NULL DEFAULT 'pending',
    is_used BOOLEAN NOT NULL DEFAULT false,
    used_at TIMESTAMPTZ NULL,
    expires_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_reservations_student_date_meal ON meal.reservations(student_id, reservation_date, meal_time);
CREATE UNIQUE INDEX idx_unique_active_reservation ON meal.reservations(student_id, reservation_date, meal_time) WHERE status IN ('pending', 'confirmed');
CREATE INDEX idx_reservations_qr_validation ON meal.reservations(cafeteria_id, reservation_date, meal_time, student_id) WHERE status = 'confirmed' AND is_used = false;
CREATE INDEX idx_reservations_batch ON meal.reservations(batch_id) WHERE batch_id IS NOT NULL;
CREATE INDEX idx_reservations_pending_expires ON meal.reservations(expires_at) WHERE status = 'pending';
CREATE INDEX idx_reservations_expired_cleanup ON meal.reservations(expires_at) WHERE status = 'expired';

CREATE TABLE IF NOT EXISTS meal.monthly_menus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    year SMALLINT NOT NULL,
    month SMALLINT NOT NULL CHECK (month BETWEEN 1 AND 12),
    menu_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_year_month UNIQUE(year, month)
);
CREATE INDEX idx_monthly_menus_year_month ON meal.monthly_menus(year, month);

CREATE TABLE meal.closed_days (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL UNIQUE,
    reason VARCHAR(255) NOT NULL,
    semester VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_closed_days_date ON meal.closed_days(date);

-- +goose Down
DROP TABLE IF EXISTS meal.closed_days;
DROP TABLE IF EXISTS meal.monthly_menus;
DROP TABLE IF EXISTS meal.reservations;
DROP TYPE IF EXISTS meal.reservation_status_enum;
DROP TYPE IF EXISTS meal.menu_type_enum;
DROP TYPE IF EXISTS meal.meal_time_enum;
DROP TABLE IF EXISTS meal.students_view;
DROP TABLE IF EXISTS meal.cafeterias;
