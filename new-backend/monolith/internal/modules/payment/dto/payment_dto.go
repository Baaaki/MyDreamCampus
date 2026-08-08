// Package dto carries the JSON contract of payment's internal endpoints.
// The service layer keeps its own tag-free request types; these mirror the
// field names meal already sends, so the wire format is unchanged from the
// in-process adapter's point of view.
package dto

// InitiatePaymentRequest is the body of POST /internal/payments/initiate.
type InitiatePaymentRequest struct {
	ReferenceID string  `json:"reference_id" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	Currency    string  `json:"currency" binding:"required"`
	Description string  `json:"description"`
	StudentID   string  `json:"student_id"`
}

// InitiatePaymentResponse is returned by POST /internal/payments/initiate.
type InitiatePaymentResponse struct {
	PaymentID  string  `json:"payment_id"`
	PaymentURL string  `json:"payment_url"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	ExpiresAt  string  `json:"expires_at"`
}

// RefundRequest is the body of POST /internal/payments/refund.
type RefundRequest struct {
	ReferenceID string  `json:"reference_id" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	Currency    string  `json:"currency" binding:"required"`
	Reason      string  `json:"reason"`
}

// RefundResponse is returned by POST /internal/payments/refund.
type RefundResponse struct {
	RefundID string  `json:"refund_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Status   string  `json:"status"`
	Message  string  `json:"message,omitempty"`
}
