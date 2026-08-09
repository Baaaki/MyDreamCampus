package contracts

// InitiatePaymentRequest is the body of POST /internal/payments/initiate.
// The binding tags are for payment's handler; meal only marshals the type.
type InitiatePaymentRequest struct {
	ReferenceID string  `json:"reference_id" binding:"required"` // "res_uuid" or "bat_uuid"
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
	ReferenceID string  `json:"reference_id" binding:"required"` // reservation ID
	Amount      float64 `json:"amount" binding:"required"`
	Currency    string  `json:"currency" binding:"required"`
	Reason      string  `json:"reason"`
}

// RefundResponse is returned by POST /internal/payments/refund.
type RefundResponse struct {
	RefundID string  `json:"refund_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Status   string  `json:"status"` // "completed", "failed", "pending"
	Message  string  `json:"message,omitempty"`
}
