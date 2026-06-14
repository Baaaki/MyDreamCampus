-- name: GetActiveCafeterias :many
SELECT id, name, location, has_vegan_menu, serves_dinner, is_active, created_at, updated_at
FROM meal.cafeterias
WHERE is_active = true
ORDER BY name ASC;

-- name: GetAllCafeterias :many
SELECT id, name, location, has_vegan_menu, serves_dinner, is_active, created_at, updated_at
FROM meal.cafeterias
ORDER BY name ASC;

-- name: GetCafeteriaByID :one
SELECT id, name, location, has_vegan_menu, serves_dinner, is_active, created_at, updated_at
FROM meal.cafeterias
WHERE id = $1;

-- name: CreateCafeteria :one
INSERT INTO meal.cafeterias (name, location, has_vegan_menu, serves_dinner, is_active)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, location, has_vegan_menu, serves_dinner, is_active, created_at, updated_at;

-- name: UpdateCafeteria :one
UPDATE meal.cafeterias
SET name = $2, location = $3, has_vegan_menu = $4, serves_dinner = $5, is_active = $6, updated_at = NOW()
WHERE id = $1
RETURNING id, name, location, has_vegan_menu, serves_dinner, is_active, created_at, updated_at;

-- name: DeactivateCafeteria :one
UPDATE meal.cafeterias
SET is_active = false, updated_at = NOW()
WHERE id = $1
RETURNING id;
