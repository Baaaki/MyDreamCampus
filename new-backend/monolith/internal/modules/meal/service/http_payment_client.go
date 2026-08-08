package service

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/dto"
	"github.com/baaaki/mydreamcampus/shared/client"
)

// HTTPPaymentClient reaches the payment service over internal REST. The
// wire types are meal's own payment DTOs, which is what PaymentAdapter
// already hands the in-process service.
type HTTPPaymentClient struct {
	base *client.Base
}

func NewHTTPPaymentClient(base *client.Base) *HTTPPaymentClient {
	return &HTTPPaymentClient{base: base}
}

func (c *HTTPPaymentClient) InitiatePayment(ctx context.Context, req dto.InitiatePaymentRequest) (*dto.InitiatePaymentResponse, error) {
	var resp dto.InitiatePaymentResponse
	if err := c.base.Post(ctx, "/internal/payments/initiate", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *HTTPPaymentClient) RequestRefund(ctx context.Context, req dto.RefundRequest) (*dto.RefundResponse, error) {
	var resp dto.RefundResponse
	if err := c.base.Post(ctx, "/internal/payments/refund", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

var _ PaymentClient = (*HTTPPaymentClient)(nil)
