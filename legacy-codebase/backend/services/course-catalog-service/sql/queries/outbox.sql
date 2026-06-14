-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (event_type, routing_key, payload)
VALUES ($1, $2, $3)
RETURNING id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message;

-- name: GetPendingEvents :many
SELECT id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message
FROM outbox_events
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkEventProcessed :exec
UPDATE outbox_events
SET status = 'processed', processed_at = NOW()
WHERE id = $1;

-- name: MarkEventFailed :exec
UPDATE outbox_events
SET status = 'failed', retry_count = retry_count + 1, error_message = $2
WHERE id = $1;

-- name: GetFailedEventsForRetry :many
SELECT id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message
FROM outbox_events
WHERE status = 'failed' AND retry_count < max_retries
ORDER BY created_at ASC
LIMIT $1;
