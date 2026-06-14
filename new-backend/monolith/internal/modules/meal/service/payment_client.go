package service

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/dto"
)

// PaymentClient defines the interface for interacting with the payment service.
// In the modular monolith, this will be implemented by an in-process adapter or the actual payment service.
type PaymentClient interface {
	InitiatePayment(ctx context.Context, req dto.InitiatePaymentRequest) (*dto.InitiatePaymentResponse, error)
	RequestRefund(ctx context.Context, req dto.RefundRequest) (*dto.RefundResponse, error)
}
