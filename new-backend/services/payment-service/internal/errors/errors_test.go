package errors

import (
	"net/http"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentErrors_HTTPStatusesAndCodes(t *testing.T) {
	tests := []struct {
		err          error
		expectedCode string
		expectedHTTP int
	}{
		{ErrPaymentNotFound, "PAYMENT_NOT_FOUND", http.StatusNotFound},
		{ErrPaymentNotPending, "PAYMENT_NOT_PENDING", http.StatusConflict},
		{ErrPaymentExpired, "PAYMENT_EXPIRED", http.StatusConflict},
		{ErrRefundNotAllowed, "REFUND_NOT_ALLOWED", http.StatusConflict},
		{ErrRefundTooLarge, "REFUND_TOO_LARGE", http.StatusConflict},
		{ErrInvalidInitiate, "INVALID_PAYMENT_REQUEST", http.StatusBadRequest},
		{ErrInvalidCardNumber, "INVALID_CARD_NUMBER", http.StatusUnprocessableEntity},
		{ErrInvalidExpiry, "INVALID_CARD_EXPIRY", http.StatusUnprocessableEntity},
		{ErrCardExpired, "CARD_EXPIRED", http.StatusUnprocessableEntity},
		{ErrInvalidCVC, "INVALID_CVC", http.StatusUnprocessableEntity},
		{ErrInvalidCardholder, "INVALID_CARDHOLDER", http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(tt.expectedCode, func(t *testing.T) {
			appErr, ok := sharedErrors.As(tt.err)
			require.True(t, ok, "must be AppError")
			assert.Equal(t, tt.expectedCode, appErr.Code)
			assert.Equal(t, tt.expectedHTTP, appErr.HTTPStatus)
		})
	}
}
