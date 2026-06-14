-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (event_type, aggregate_id, payload, created_at, status)
VALUES ($1, $2, $3, NOW(), 'pending')
RETURNING id, event_type, aggregate_id, payload, created_at, processed_at, status;

-- name: GetPendingOutboxEvents :many
SELECT id, event_type, aggregate_id, payload, created_at, processed_at, status
FROM outbox_events
WHERE status = 'pending'
ORDER BY created_at
LIMIT $1;

-- name: MarkOutboxEventProcessed :exec
UPDATE outbox_events
SET status = 'processed', processed_at = NOW()
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET status = 'failed'
WHERE id = $1;
