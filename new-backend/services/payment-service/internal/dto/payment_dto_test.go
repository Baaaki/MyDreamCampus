package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmPaymentRequest_JSONSerialization(t *testing.T) {
	raw := `{
		"card_number": "4111111111111111",
		"exp_month": 12,
		"exp_year": 2028,
		"cvc": "123",
		"cardholder_name": "Can Yilmaz"
	}`

	var req ConfirmPaymentRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Equal(t, "4111111111111111", req.CardNumber)
	assert.Equal(t, 12, req.ExpMonth)
	assert.Equal(t, 2028, req.ExpYear)
	assert.Equal(t, "123", req.CVC)
	assert.Equal(t, "Can Yilmaz", req.CardholderName)
}

func TestPaymentResponse_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	brand := "Visa"
	last4 := "1111"

	resp := PaymentResponse{
		ID:          "pay-12345",
		Status:      "completed",
		Amount:      45.50,
		Currency:    "TRY",
		CardBrand:   &brand,
		CardLast4:   &last4,
		ExpiresAt:   now.Add(15 * time.Minute),
		CompletedAt: &now,
	}

	bytes, err := json.Marshal(resp)
	require.NoError(t, err)

	var unmarshaled PaymentResponse
	err = json.Unmarshal(bytes, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, resp.ID, unmarshaled.ID)
	assert.Equal(t, resp.Status, unmarshaled.Status)
	assert.Equal(t, resp.Amount, unmarshaled.Amount)
	assert.Equal(t, resp.Currency, unmarshaled.Currency)
	assert.Equal(t, *resp.CardBrand, *unmarshaled.CardBrand)
	assert.Equal(t, *resp.CardLast4, *unmarshaled.CardLast4)
}
