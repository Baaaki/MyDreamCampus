-- name: CreateProcessedEvent :exec
INSERT INTO student.processed_events (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: IsEventProcessed :one
SELECT EXISTS(SELECT 1 FROM student.processed_events WHERE event_id = $1);

-- name: DeleteOldProcessedEvents :execrows
-- Retention: the dedup ledger only has to outlive redelivery, not the row it
-- guarded. Retention window comes from the caller, not the query.
DELETE FROM student.processed_events
WHERE processed_at < $1;
