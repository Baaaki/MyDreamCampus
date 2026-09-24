package service

import (
	"github.com/baaaki/mydreamcampus/payment/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
)

// Wire-contract payload builders for payment outbox events. The consumer is
// meal (dto.PaymentCompletedEventData / PaymentFailedEventData): it routes
// on reference_id's "res_" / "bat_" prefix, so a renamed key leaves every
// paid reservation pending until it expires. No card data goes in.

func buildPaymentCompletedPayload(p db.Payment) (map[string]any, error) {
	amount, err := utils.PgNumericToFloat64(p.Amount)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"payment_id":   utils.PgtypeToUUIDString(p.ID),
		"reference_id": p.ReferenceID,
		"amount":       amount,
		"currency":     p.Currency,
	}, nil
}

func buildPaymentFailedPayload(p db.Payment, reason string) map[string]any {
	return map[string]any{
		"payment_id":   utils.PgtypeToUUIDString(p.ID),
		"reference_id": p.ReferenceID,
		"reason":       reason,
	}
}
