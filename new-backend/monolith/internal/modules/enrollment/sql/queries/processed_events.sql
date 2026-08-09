-- name: CreateProcessedEvent :one
INSERT INTO enrollment.processed_events (event_id, event_type, processed_at)
VALUES ($1, $2, NOW())
RETURNING event_id, event_type, processed_at;

-- name: IsEventProcessed :one
SELECT EXISTS(
    SELECT 1 FROM enrollment.processed_events WHERE event_id = $1
) as processed;

-- name: DeleteOldProcessedEvents :execrows
-- Retention: the dedup ledger only has to outlive redelivery, not the row it
-- guarded. Retention window comes from the caller, not the query.
DELETE FROM enrollment.processed_events
WHERE processed_at < $1;
