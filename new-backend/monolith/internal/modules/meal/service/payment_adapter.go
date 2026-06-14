package service

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/dto"
	paymentSvc "github.com/baaaki/mydreamcampus/monolith/internal/modules/payment/service"
)

type PaymentAdapter struct {
	paymentService *paymentSvc.PaymentService
}

func NewPaymentAdapter(paymentService *paymentSvc.PaymentService) *PaymentAdapter {
	return &PaymentAdapter{
		paymentService: paymentService,
	}
}

func (a *PaymentAdapter) InitiatePayment(ctx context.Context, req dto.InitiatePaymentRequest) (*dto.InitiatePaymentResponse, error) {
	paymentReq := paymentSvc.InitiatePaymentRequest{
		ReferenceID: req.ReferenceID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		StudentID:   req.StudentID,
	}

	resp, err := a.paymentService.InitiatePayment(ctx, paymentReq)
	if err != nil {
		return nil, err
	}

	return &dto.InitiatePaymentResponse{
		PaymentID:  resp.PaymentID,
		PaymentURL: resp.PaymentURL,
		Amount:     resp.Amount,
		Currency:   resp.Currency,
		ExpiresAt:  resp.ExpiresAt,
	}, nil
}

func (a *PaymentAdapter) RequestRefund(ctx context.Context, req dto.RefundRequest) (*dto.RefundResponse, error) {
	refundReq := paymentSvc.RefundRequest{
		ReferenceID: req.ReferenceID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Reason:      req.Reason,
	}

	resp, err := a.paymentService.RequestRefund(ctx, refundReq)
	if err != nil {
		return nil, err
	}

	return &dto.RefundResponse{
		RefundID: resp.RefundID,
		Amount:   resp.Amount,
		Currency: resp.Currency,
		Status:   resp.Status,
		Message:  resp.Message,
	}, nil
}
