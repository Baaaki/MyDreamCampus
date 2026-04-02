-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (event_type, routing_key, payload)
VALUES ($1, $2, $3)
RETURNING id, event_type, routing_key, payload, processed, created_at, processed_at;

-- name: GetUnprocessedEvents :many
SELECT id, event_type, routing_key, payload, processed, created_at, processed_at
FROM outbox_events
WHERE processed = false
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkEventProcessed :exec
UPDATE outbox_events
SET processed = true, processed_at = NOW()
WHERE id = $1;
