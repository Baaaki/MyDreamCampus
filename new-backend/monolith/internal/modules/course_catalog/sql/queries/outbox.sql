-- name: CreateOutboxEvent :one
INSERT INTO course_catalog.outbox_events (event_type, routing_key, payload)
VALUES ($1, $2, $3)
RETURNING id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message;

-- name: GetPendingEvents :many
SELECT id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message
FROM course_catalog.outbox_events
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkEventProcessed :exec
UPDATE course_catalog.outbox_events
SET status = 'processed', processed_at = NOW()
WHERE id = $1;

-- name: MarkEventFailed :exec
UPDATE course_catalog.outbox_events
SET status = 'failed', retry_count = retry_count + 1, error_message = $2
WHERE id = $1;

-- name: GetFailedEventsForRetry :many
SELECT id, event_type, routing_key, payload, status, retry_count, max_retries, created_at, processed_at, error_message
FROM course_catalog.outbox_events
WHERE status = 'failed' AND retry_count < max_retries
ORDER BY created_at ASC
LIMIT $1;

-- name: ResetFailedOutboxEvent :exec
-- Used by the generic eventbus.OutboxWorker to flip a failed event back
-- to pending so the next poll retries it.
UPDATE course_catalog.outbox_events
SET status = 'pending', error_message = NULL
WHERE id = $1;
