-- name: InsertAuditLog :one
INSERT INTO audit_log (service, actor_id, actor_role, action, resource_type, resource_id, details)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListAuditLog :many
SELECT * FROM audit_log
WHERE
    (sqlc.narg('service')::VARCHAR IS NULL OR service = sqlc.narg('service')) AND
    (sqlc.narg('action')::VARCHAR IS NULL OR action = sqlc.narg('action')) AND
    (sqlc.narg('actor_id')::UUID IS NULL OR actor_id = sqlc.narg('actor_id'))
ORDER BY timestamp DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLog :one
SELECT COUNT(*) FROM audit_log
WHERE
    (sqlc.narg('service')::VARCHAR IS NULL OR service = sqlc.narg('service')) AND
    (sqlc.narg('action')::VARCHAR IS NULL OR action = sqlc.narg('action')) AND
    (sqlc.narg('actor_id')::UUID IS NULL OR actor_id = sqlc.narg('actor_id'));
