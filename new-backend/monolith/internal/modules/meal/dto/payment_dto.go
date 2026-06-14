package dto

// Payment Service DTOs for HTTP requests

// InitiatePaymentRequest represents request to payment service
type InitiatePaymentRequest struct {
	ReferenceID string  `json:"reference_id"` // "res_uuid" or "bat_uuid"
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	StudentID   string  `json:"student_id"`
}

// InitiatePaymentResponse represents response from payment service
type InitiatePaymentResponse struct {
	PaymentID  string  `json:"payment_id"`
	PaymentURL string  `json:"payment_url"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	ExpiresAt  string  `json:"expires_at"`
}

// RefundRequest represents refund request to payment service
type RefundRequest struct {
	ReferenceID  string  `json:"reference_id"` // reservation ID
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Reason       string  `json:"reason"`
}

// RefundResponse represents refund response from payment service
type RefundResponse struct {
	RefundID     string  `json:"refund_id"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Status       string  `json:"status"` // "completed", "failed", "pending"
	Message      string  `json:"message,omitempty"`
}
