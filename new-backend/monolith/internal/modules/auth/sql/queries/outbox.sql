-- name: CreateOutboxEvent :one
INSERT INTO auth.outbox_events (event_type, routing_key, payload)
VALUES ($1, $2, $3)
RETURNING id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message;

-- name: GetPendingOutboxEvents :many
SELECT id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message
FROM auth.outbox_events
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkOutboxEventProcessed :exec
UPDATE auth.outbox_events
SET status = 'processed', processed_at = NOW()
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE auth.outbox_events
SET status = 'failed', retry_count = retry_count + 1, error_message = $2
WHERE id = $1;

-- name: GetFailedOutboxEvents :many
SELECT id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message
FROM auth.outbox_events
WHERE status = 'failed' AND retry_count < max_retries
ORDER BY created_at ASC
LIMIT $1;

-- name: ResetFailedOutboxEvent :exec
UPDATE auth.outbox_events
SET status = 'pending', error_message = NULL
WHERE id = $1;
