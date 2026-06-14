-- name: CreateReservation :one
INSERT INTO meal.reservations (batch_id, student_id, cafeteria_id, reservation_date, meal_time, menu_type, status, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, batch_id, student_id, cafeteria_id, reservation_date, meal_time, menu_type, status, is_used, used_at, expires_at, created_at, updated_at;

-- name: GetReservationByID :one
SELECT r.id, r.batch_id, r.student_id, r.cafeteria_id, r.reservation_date, r.meal_time, r.menu_type, r.status, r.is_used, r.used_at, r.expires_at, r.created_at, r.updated_at,
       c.name as cafeteria_name, c.location as cafeteria_location
FROM meal.reservations r
JOIN meal.cafeterias c ON r.cafeteria_id = c.id
WHERE r.id = $1;

-- name: CheckActiveReservation :one
SELECT id, status
FROM meal.reservations
WHERE student_id = $1 AND reservation_date = $2 AND meal_time = $3
  AND status IN ('pending', 'confirmed')
LIMIT 1;

-- name: CheckActiveReservationsForSlots :many
-- Batch variant of CheckActiveReservation. Accepts parallel arrays of dates and
-- meal times and returns any existing active rows for (student_id, date, meal).
SELECT r.id, r.reservation_date, r.meal_time, r.status, c.name AS cafeteria_name
FROM meal.reservations r
JOIN meal.cafeterias c ON r.cafeteria_id = c.id
WHERE r.student_id = $1
  AND r.status IN ('pending', 'confirmed')
  AND (r.reservation_date, r.meal_time::text) IN (
    SELECT unnest(@dates::date[]), unnest(@meal_times::text[])
  );

-- name: GetStudentReservations :many
SELECT r.id, r.batch_id, r.student_id, r.cafeteria_id, r.reservation_date, r.meal_time, r.menu_type, r.status, r.is_used, r.used_at, r.expires_at, r.created_at, r.updated_at,
       c.name as cafeteria_name, c.location as cafeteria_location
FROM meal.reservations r
JOIN meal.cafeterias c ON r.cafeteria_id = c.id
WHERE r.student_id = $1
ORDER BY r.reservation_date DESC, r.meal_time ASC;

-- name: GetStudentReservationsFiltered :many
SELECT r.id, r.batch_id, r.student_id, r.cafeteria_id, r.reservation_date, r.meal_time, r.menu_type, r.status, r.is_used, r.used_at, r.expires_at, r.created_at, r.updated_at,
       c.name as cafeteria_name, c.location as cafeteria_location
FROM meal.reservations r
JOIN meal.cafeterias c ON r.cafeteria_id = c.id
WHERE r.student_id = $1
  AND ($2::date IS NULL OR r.reservation_date >= $2)
  AND ($3::date IS NULL OR r.reservation_date <= $3)
  AND (sqlc.narg('status')::meal.reservation_status_enum IS NULL OR r.status = sqlc.narg('status'))
ORDER BY r.reservation_date DESC, r.meal_time ASC
LIMIT NULLIF($4, 0) OFFSET $5;

-- name: CountStudentReservationsFiltered :one
SELECT COUNT(*) as total
FROM meal.reservations r
WHERE r.student_id = $1
  AND ($2::date IS NULL OR r.reservation_date >= $2)
  AND ($3::date IS NULL OR r.reservation_date <= $3)
  AND (sqlc.narg('status')::meal.reservation_status_enum IS NULL OR r.status = sqlc.narg('status'));

-- name: UpdateReservationByID :one
UPDATE meal.reservations
SET status = $2, expires_at = $3, updated_at = NOW()
WHERE id = $1
RETURNING id, batch_id, student_id, cafeteria_id, reservation_date, meal_time, menu_type, status, is_used, used_at, expires_at, created_at, updated_at;

-- name: UpdateReservationsByBatchID :exec
UPDATE meal.reservations
SET status = $2, expires_at = $3, updated_at = NOW()
WHERE batch_id = $1;

-- name: MarkReservationUsed :one
UPDATE meal.reservations
SET is_used = true, used_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING id, batch_id, student_id, cafeteria_id, reservation_date, meal_time, menu_type, status, is_used, used_at, expires_at, created_at, updated_at;

-- name: FindReservationForQR :one
SELECT r.id, r.batch_id, r.student_id, r.cafeteria_id, r.reservation_date, r.meal_time, r.menu_type, r.status, r.is_used, r.used_at, r.expires_at, r.created_at, r.updated_at,
       c.name as cafeteria_name, c.location as cafeteria_location
FROM meal.reservations r
JOIN meal.cafeterias c ON r.cafeteria_id = c.id
WHERE r.cafeteria_id = $1
  AND r.reservation_date = $2
  AND r.meal_time = $3
  AND r.student_id = $4
  AND r.status = 'confirmed'
  AND r.is_used = false
LIMIT 1;

-- name: ExpirePendingReservations :exec
WITH expired_batch AS (
    SELECT id FROM meal.reservations
    WHERE status = 'pending' AND expires_at < NOW()
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE meal.reservations
SET status = 'expired', updated_at = NOW()
WHERE id IN (SELECT id FROM expired_batch);

-- name: CleanupExpiredReservations :exec
WITH cleanup_batch AS (
    SELECT id FROM meal.reservations
    WHERE status = 'expired' AND expires_at < NOW() - INTERVAL '7 days'
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
DELETE FROM meal.reservations
WHERE id IN (SELECT id FROM cleanup_batch);

-- name: CancelReservation :one
UPDATE meal.reservations
SET status = 'cancelled', updated_at = NOW()
WHERE id = $1
RETURNING id, batch_id, student_id, cafeteria_id, reservation_date, meal_time, menu_type, status, is_used, used_at, expires_at, created_at, updated_at;
