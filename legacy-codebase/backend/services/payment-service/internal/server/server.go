package server

import (
	"context"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/shared/clock"
	pb "github.com/baaaki/mydreamcampus/payment-service/proto"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PaymentServer implements the gRPC PaymentService
// This is a mock implementation that always returns success
type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	logger *zap.Logger
}

// NewPaymentServer creates a new PaymentServer instance
func NewPaymentServer(logger *zap.Logger) *PaymentServer {
	return &PaymentServer{
		logger: logger,
	}
}

// InitiatePayment handles payment initiation requests
// MOCK: Always returns success for development/testing
func (s *PaymentServer) InitiatePayment(ctx context.Context, req *pb.InitiatePaymentRequest) (*pb.InitiatePaymentResponse, error) {
	s.logger.Info("MOCK: Processing payment initiation",
		zap.String("reference_id", req.ReferenceId),
		zap.Float64("amount", req.Amount),
		zap.String("currency", req.Currency),
		zap.String("student_id", req.StudentId),
		zap.String("description", req.Description),
	)

	// Generate mock payment ID
	paymentID := fmt.Sprintf("pay_%s", uuid.New().String()[:8])

	// Generate mock payment URL
	paymentURL := fmt.Sprintf("https://mock-payment.mydreamcampus.com/pay/%s", paymentID)

	// Set expiration time (30 minutes from now)
	expiresAt := clock.Now().Add(30 * time.Minute).Format(time.RFC3339)

	s.logger.Info("MOCK: Payment initiated successfully",
		zap.String("payment_id", paymentID),
		zap.String("payment_url", paymentURL),
	)

	return &pb.InitiatePaymentResponse{
		Success:    true,
		PaymentId:  paymentID,
		PaymentUrl: paymentURL,
		Amount:     req.Amount,
		Currency:   req.Currency,
		ExpiresAt:  expiresAt,
	}, nil
}

// RequestRefund handles refund requests
// MOCK: Always returns success for development/testing
func (s *PaymentServer) RequestRefund(ctx context.Context, req *pb.RefundRequest) (*pb.RefundResponse, error) {
	s.logger.Info("MOCK: Processing refund request",
		zap.String("reference_id", req.ReferenceId),
		zap.Float64("amount", req.Amount),
		zap.String("currency", req.Currency),
		zap.String("reason", req.Reason),
	)

	// Generate mock refund ID
	refundID := fmt.Sprintf("ref_%s", uuid.New().String()[:8])

	s.logger.Info("MOCK: Refund completed successfully",
		zap.String("refund_id", refundID),
	)

	return &pb.RefundResponse{
		Success:  true,
		RefundId: refundID,
		Amount:   req.Amount,
		Currency: req.Currency,
		Status:   "completed",
		Message:  "Refund processed successfully",
	}, nil
}
