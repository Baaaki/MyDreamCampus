-- +goose Up
-- Create ENUM types
CREATE TYPE meal_time_enum AS ENUM ('lunch', 'dinner');
CREATE TYPE menu_type_enum AS ENUM ('normal', 'vegan');
CREATE TYPE reservation_status_enum AS ENUM ('pending', 'confirmed', 'cancelled', 'expired');

-- Create reservations table
CREATE TABLE IF NOT EXISTS reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NULL,
    student_id UUID NOT NULL REFERENCES students_cache(id) ON DELETE CASCADE,
    cafeteria_id UUID NOT NULL REFERENCES cafeterias(id),
    reservation_date DATE NOT NULL,
    meal_time meal_time_enum NOT NULL,
    menu_type menu_type_enum NOT NULL DEFAULT 'normal',
    status reservation_status_enum NOT NULL DEFAULT 'pending',
    is_used BOOLEAN NOT NULL DEFAULT false,
    used_at TIMESTAMPTZ NULL,
    expires_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for query optimization
-- Öğrencinin belirli gün/öğün rezervasyonu var mı?
CREATE INDEX idx_reservations_student_date_meal
ON reservations(student_id, reservation_date, meal_time);

-- Aktif rezervasyonlar için unique constraint (aynı gün/öğüne birden fazla aktif rezervasyon engellenir)
CREATE UNIQUE INDEX idx_unique_active_reservation
ON reservations(student_id, reservation_date, meal_time)
WHERE status IN ('pending', 'confirmed');

-- QR doğrulama sorgusu: Belirli yemekhane/gün/öğün/öğrenci için confirmed rezervasyon
CREATE INDEX idx_reservations_qr_validation
ON reservations(cafeteria_id, reservation_date, meal_time, student_id)
WHERE status = 'confirmed' AND is_used = false;

-- Batch lookup (toplu rezervasyonlarda payment callback için)
CREATE INDEX idx_reservations_batch
ON reservations(batch_id)
WHERE batch_id IS NOT NULL;

-- Expiry job için: Pending ve süresi dolmuş rezervasyonlar
CREATE INDEX idx_reservations_pending_expires
ON reservations(expires_at)
WHERE status = 'pending';

-- Cleanup job için: Expired ve 7 günden eski rezervasyonlar
CREATE INDEX idx_reservations_expired_cleanup
ON reservations(expires_at)
WHERE status = 'expired';

-- +goose Down
DROP TABLE IF EXISTS reservations;
DROP TYPE IF EXISTS meal_time_enum;
DROP TYPE IF EXISTS menu_type_enum;
DROP TYPE IF EXISTS reservation_status_enum;
