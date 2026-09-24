// Package dto carries payment's JSON contracts. The internal endpoints'
// shapes live in shared/contracts so meal — their only caller — binds
// against the exact same struct; the student-facing ones are defined here.
package dto

import (
	"time"

	"github.com/baaaki/mydreamcampus/shared/contracts"
)

type (
	InitiatePaymentRequest  = contracts.InitiatePaymentRequest
	InitiatePaymentResponse = contracts.InitiatePaymentResponse
	RefundRequest           = contracts.RefundRequest
	RefundResponse          = contracts.RefundResponse
)

// ConfirmPaymentRequest is the card form. No binding tags: the card package
// checks every field and names the one that is wrong.
type ConfirmPaymentRequest struct {
	CardNumber     string `json:"card_number"`
	ExpMonth       int    `json:"exp_month"`
	ExpYear        int    `json:"exp_year"`
	CVC            string `json:"cvc"`
	CardholderName string `json:"cardholder_name"`
}

// PaymentResponse is a payment as its student sees it.
type PaymentResponse struct {
	ID            string     `json:"id"`
	Status        string     `json:"status"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	CardBrand     *string    `json:"card_brand"`
	CardLast4     *string    `json:"card_last4"`
	FailureReason *string    `json:"failure_reason"`
	ExpiresAt     time.Time  `json:"expires_at"`
	CompletedAt   *time.Time `json:"completed_at"`
}
