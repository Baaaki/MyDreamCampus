-- name: CreateProcessedEvent :exec
INSERT INTO attendance.processed_events (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: CheckEventProcessed :one
SELECT COUNT(*) as count
FROM attendance.processed_events
WHERE event_id = $1;
