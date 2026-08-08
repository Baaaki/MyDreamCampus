-- name: CreateOutboxEvent :one
INSERT INTO student.outbox_events (event_type, routing_key, payload, correlation_id)
VALUES ($1, $2, $3, $4)
RETURNING id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message, correlation_id;

-- name: GetPendingOutboxEvents :many
SELECT id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message, correlation_id
FROM student.outbox_events
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkOutboxEventProcessed :exec
UPDATE student.outbox_events
SET status = 'processed', processed_at = NOW()
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE student.outbox_events
SET status = 'failed', retry_count = retry_count + 1, error_message = $2
WHERE id = $1;

-- name: GetFailedOutboxEvents :many
SELECT id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message, correlation_id
FROM student.outbox_events
WHERE status = 'failed' AND retry_count < max_retries
ORDER BY created_at ASC
LIMIT $1;

-- name: ResetFailedOutboxEvent :exec
UPDATE student.outbox_events
SET status = 'pending', error_message = NULL
WHERE id = $1;
