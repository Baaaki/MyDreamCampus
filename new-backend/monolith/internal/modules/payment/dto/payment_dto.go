// Package dto carries the JSON contract of payment's internal endpoints.
// The service layer keeps its own tag-free request types; the wire shapes
// live in shared/contracts so meal — the only caller — binds against the
// exact same struct.
package dto

import "github.com/baaaki/mydreamcampus/shared/contracts"

type (
	InitiatePaymentRequest  = contracts.InitiatePaymentRequest
	InitiatePaymentResponse = contracts.InitiatePaymentResponse
	RefundRequest           = contracts.RefundRequest
	RefundResponse          = contracts.RefundResponse
)
