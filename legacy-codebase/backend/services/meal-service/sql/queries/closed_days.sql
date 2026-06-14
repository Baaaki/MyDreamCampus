-- name: CreateClosedDay :one
INSERT INTO closed_days (date, reason, semester)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteClosedDay :exec
DELETE FROM closed_days WHERE id = $1;

-- name: ListClosedDays :many
SELECT * FROM closed_days
WHERE ($1::date IS NULL OR date >= $1)
  AND ($2::date IS NULL OR date <= $2)
ORDER BY date ASC;

-- name: IsDateClosed :one
SELECT EXISTS(
    SELECT 1 FROM closed_days WHERE date = $1
) AS is_closed;

-- name: GetClosedDaysByDateRange :many
SELECT * FROM closed_days
WHERE date >= $1 AND date <= $2
ORDER BY date ASC;

-- name: DeleteClosedDaysBySemester :exec
DELETE FROM closed_days WHERE semester = $1;
