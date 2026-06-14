-- name: CreateProcessedEvent :exec
INSERT INTO meal.processed_events (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: IsEventProcessed :one
SELECT EXISTS(SELECT 1 FROM meal.processed_events WHERE event_id = $1);

-- name: CleanupOldProcessedEvents :exec
DELETE FROM meal.processed_events
WHERE processed_at < NOW() - INTERVAL '30 days';
