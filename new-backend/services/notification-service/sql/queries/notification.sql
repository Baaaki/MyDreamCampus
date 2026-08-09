-- name: CreateDeliveryLog :one
INSERT INTO delivery_log (
    event_id, event_type, channel, recipient, template, status, error, sent_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: UpdateDeliveryLogStatus :exec
UPDATE delivery_log
SET status = $2, error = $3, sent_at = $4
WHERE id = $1;

-- name: GetDeliveryLogByEventID :many
SELECT * FROM delivery_log
WHERE event_id = $1;

-- name: MarkEventProcessed :exec
INSERT INTO processed_events (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: IsEventProcessed :one
SELECT EXISTS (
    SELECT 1 FROM processed_events
    WHERE event_id = $1
);
