-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (aggregate_id, aggregate_type, event_type, payload, max_retries)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, aggregate_id, aggregate_type, event_type, payload, status, retry_count, max_retries, next_retry_at, last_error, created_at, published_at;

-- name: GetPendingOutboxEvents :many
SELECT id, aggregate_id, aggregate_type, event_type, payload, status, retry_count, max_retries, next_retry_at, last_error, created_at, published_at
FROM outbox_events
WHERE status = 'pending'
  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events
SET status = 'published', published_at = NOW()
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET status = 'failed', last_error = $2, retry_count = retry_count + 1
WHERE id = $1;

-- name: UpdateOutboxEventRetry :exec
UPDATE outbox_events
SET retry_count = retry_count + 1, next_retry_at = $2, last_error = $3
WHERE id = $1;

-- name: GetFailedOutboxEvents :many
SELECT id, aggregate_id, aggregate_type, event_type, payload, status, retry_count, max_retries, next_retry_at, last_error, created_at, published_at
FROM outbox_events
WHERE status = 'failed'
ORDER BY created_at DESC;
