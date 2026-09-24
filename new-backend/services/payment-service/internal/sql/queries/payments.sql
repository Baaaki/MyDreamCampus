-- name: CreatePayment :one
-- A retried initiate hits the unique reference_id and gets the existing row
-- back: the no-op DO UPDATE is what makes RETURNING yield it.
INSERT INTO payment.payments (reference_id, student_id, amount, currency, description, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (reference_id) DO UPDATE SET reference_id = EXCLUDED.reference_id
RETURNING *;

-- name: GetPaymentByReferenceID :one
SELECT * FROM payment.payments
WHERE reference_id = $1;

-- name: RefundPayment :one
-- The guard sits in the WHERE clause so two concurrent refunds cannot
-- together return more than was paid.
UPDATE payment.payments
SET refunded_amount = refunded_amount + sqlc.arg(refund)::numeric,
    status = CASE
        WHEN refunded_amount + sqlc.arg(refund)::numeric >= amount THEN 'refunded'::payment.payment_status_enum
        ELSE status
    END,
    updated_at = NOW()
WHERE reference_id = sqlc.arg(reference_id)
  AND status = 'completed'
  AND refunded_amount + sqlc.arg(refund)::numeric <= amount
RETURNING *;

-- name: ExpireOverduePayments :execrows
UPDATE payment.payments
SET status = 'expired', updated_at = NOW()
WHERE status = 'pending' AND expires_at <= $1;
