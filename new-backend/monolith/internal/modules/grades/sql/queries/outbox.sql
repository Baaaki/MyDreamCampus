-- name: CreateOutboxEvent :one
INSERT INTO grades.outbox_events (
    event_type, routing_key, payload, status, retry_count, max_retries
) VALUES (
    $1, $2, $3, 'pending', 0, 3
)
RETURNING *;

-- name: GetPendingOutboxEvents :many
SELECT * FROM grades.outbox_events
WHERE status = 'pending'
ORDER BY created_at
LIMIT $1;

-- name: MarkOutboxEventProcessed :exec
UPDATE grades.outbox_events
SET status = 'processed', processed_at = NOW()
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE grades.outbox_events
SET status = 'failed',
    retry_count = retry_count + 1,
    error_message = $2,
    processed_at = NOW()
WHERE id = $1;

-- name: RetryFailedOutboxEvent :exec
UPDATE grades.outbox_events
SET status = 'pending',
    retry_count = retry_count + 1
WHERE id = $1 AND retry_count < max_retries;
