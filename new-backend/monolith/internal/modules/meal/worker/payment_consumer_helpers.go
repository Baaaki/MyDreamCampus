package worker

import (
	"strings"

	"github.com/google/uuid"
)

// parseReferenceID extracts the underlying UUID from a payment service
// reference_id of shape "<prefix>_<uuid>" and reports whether the reference
// addresses a batch reservation.
//
// Wire contract: payment-service emits reference_id with prefix "res_" for a
// single reservation and "bat_" for a batch. Both prefixes are stripped.
// Anything else (or a missing prefix) is forwarded to uuid.Parse — it will
// return an error rather than silently coercing.
//
// The "bat_" prefix is the *only* signal for batch routing in
// HandlePaymentCompleted / HandlePaymentFailed; do not loosen it.
func parseReferenceID(ref string) (id uuid.UUID, isBatch bool, err error) {
	isBatch = strings.HasPrefix(ref, "bat_")

	stripped := strings.TrimPrefix(ref, "res_")
	stripped = strings.TrimPrefix(stripped, "bat_")

	id, err = uuid.Parse(stripped)
	return id, isBatch, err
}
