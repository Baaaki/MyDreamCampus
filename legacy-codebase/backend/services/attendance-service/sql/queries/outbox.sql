-- name: CreateOutboxEvent :exec
INSERT INTO outbox_events (event_type, routing_key, payload)
VALUES ($1, $2, $3);

-- name: GetPendingOutboxEvents :many
SELECT id, event_type, routing_key, payload, retry_count, max_retries
FROM outbox_events
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkOutboxEventProcessed :exec
UPDATE outbox_events
SET status = 'processed', processed_at = NOW()
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET retry_count = retry_count + 1,
    status = CASE
        WHEN retry_count + 1 >= max_retries THEN 'failed'::outbox_status_enum
        ELSE 'pending'::outbox_status_enum
    END,
    error_message = $2
WHERE id = $1;
