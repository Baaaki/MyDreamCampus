-- name: IsEventProcessed :one
SELECT EXISTS(SELECT 1 FROM auth.processed_events WHERE event_id = $1) AS exists;

-- name: MarkEventProcessed :exec
INSERT INTO auth.processed_events (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: DeleteOldProcessedEvents :execrows
-- Retention: the dedup ledger only has to outlive redelivery, not the row it
-- guarded. Retention window comes from the caller, not the query.
DELETE FROM auth.processed_events
WHERE processed_at < $1;
