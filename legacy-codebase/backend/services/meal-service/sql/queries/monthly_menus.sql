-- name: UpsertMonthlyMenu :one
INSERT INTO monthly_menus (year, month, menu_data)
VALUES ($1, $2, $3)
ON CONFLICT (year, month) DO UPDATE
SET menu_data = EXCLUDED.menu_data, updated_at = NOW()
RETURNING id, year, month, menu_data, created_at, updated_at;

-- name: GetMonthlyMenu :one
SELECT id, year, month, menu_data, created_at, updated_at
FROM monthly_menus
WHERE year = $1 AND month = $2;
