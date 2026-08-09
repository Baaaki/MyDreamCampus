package dto

import "github.com/baaaki/mydreamcampus/shared/contracts"

// Payment wire types — meal is the caller, payment the provider, so the
// structs live in shared/contracts and neither side owns a private copy
// that can drift.
type (
	InitiatePaymentRequest  = contracts.InitiatePaymentRequest
	InitiatePaymentResponse = contracts.InitiatePaymentResponse
	RefundRequest           = contracts.RefundRequest
	RefundResponse          = contracts.RefundResponse
)
