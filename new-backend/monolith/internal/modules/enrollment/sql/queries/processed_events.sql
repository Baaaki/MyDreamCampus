-- name: CreateProcessedEvent :one
INSERT INTO enrollment.processed_events (event_id, event_type, processed_at)
VALUES ($1, $2, NOW())
RETURNING event_id, event_type, processed_at;

-- name: IsEventProcessed :one
SELECT EXISTS(
    SELECT 1 FROM enrollment.processed_events WHERE event_id = $1
) as processed;
