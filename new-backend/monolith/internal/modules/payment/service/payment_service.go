package service

import (
	"context"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/clock"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// InitiatePaymentRequest represents the payment details
type InitiatePaymentRequest struct {
	ReferenceID string
	Amount      float64
	Currency    string
	Description string
	StudentID   string
}

// InitiatePaymentResponse represents the mock response
type InitiatePaymentResponse struct {
	PaymentID  string
	PaymentURL string
	Amount     float64
	Currency   string
	ExpiresAt  string
}

// RefundRequest represents the refund details
type RefundRequest struct {
	ReferenceID string
	Amount      float64
	Currency    string
	Reason      string
}

// RefundResponse represents the mock refund response
type RefundResponse struct {
	RefundID string
	Amount   float64
	Currency string
	Status   string
	Message  string
}

// PaymentCompletedEvent is published after a successful mock payment
type PaymentCompletedEvent struct {
	EventType string                    `json:"event_type"`
	EventID   string                    `json:"event_id"`
	Timestamp time.Time                 `json:"timestamp"`
	Data      PaymentCompletedEventData `json:"data"`
}

type PaymentCompletedEventData struct {
	PaymentID   string  `json:"payment_id"`
	ReferenceID string  `json:"reference_id"` // "res_uuid" or "bat_uuid"
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
}

// PaymentService is the in-process mock payment service
type PaymentService struct {
	publisher *rabbitmq.Publisher
	logger    *zap.Logger
}

func NewPaymentService(publisher *rabbitmq.Publisher, logger *zap.Logger) *PaymentService {
	// Ensure the exchanges and queues are declared
	publisher.DeclareExchange("payment.events")
	publisher.DeclareAndBindQueue("meal.payment_completed_queue", "payment.events", "payment.completed")
	publisher.DeclareAndBindQueue("meal.payment_failed_queue", "payment.events", "payment.failed")

	return &PaymentService{
		publisher: publisher,
		logger:    logger,
	}
}

// InitiatePayment mocks a payment initiation. It returns success and immediately publishes a payment.completed event.
func (s *PaymentService) InitiatePayment(ctx context.Context, req InitiatePaymentRequest) (*InitiatePaymentResponse, error) {
	s.logger.Info("MOCK: Processing payment initiation",
		zap.String("reference_id", req.ReferenceID),
		zap.Float64("amount", req.Amount),
		zap.String("currency", req.Currency),
		zap.String("student_id", req.StudentID),
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

	// Publish payment.completed event to mock an asynchronous webhook confirmation
	go func() {
		// Small delay to simulate async payment
		time.Sleep(2 * time.Second)

		event := PaymentCompletedEvent{
			EventType: "payment.completed",
			EventID:   uuid.New().String(),
			Timestamp: clock.Now(),
			Data: PaymentCompletedEventData{
				PaymentID:   paymentID,
				ReferenceID: req.ReferenceID,
				Amount:      req.Amount,
				Currency:    req.Currency,
			},
		}

		err := s.publisher.Publish(context.Background(), "payment.events", "payment.completed", event)
		if err != nil {
			s.logger.Error("failed to publish mock payment.completed event", zap.Error(err))
		} else {
			s.logger.Info("MOCK: Published payment.completed event", zap.String("payment_id", paymentID))
		}
	}()

	return &InitiatePaymentResponse{
		PaymentID:  paymentID,
		PaymentURL: paymentURL,
		Amount:     req.Amount,
		Currency:   req.Currency,
		ExpiresAt:  expiresAt,
	}, nil
}

// RequestRefund handles refund requests
// MOCK: Always returns success for development/testing
func (s *PaymentService) RequestRefund(ctx context.Context, req RefundRequest) (*RefundResponse, error) {
	s.logger.Info("MOCK: Processing refund request",
		zap.String("reference_id", req.ReferenceID),
		zap.Float64("amount", req.Amount),
		zap.String("currency", req.Currency),
		zap.String("reason", req.Reason),
	)

	// Generate mock refund ID
	refundID := fmt.Sprintf("ref_%s", uuid.New().String()[:8])

	s.logger.Info("MOCK: Refund completed successfully",
		zap.String("refund_id", refundID),
	)

	return &RefundResponse{
		RefundID: refundID,
		Amount:   req.Amount,
		Currency: req.Currency,
		Status:   "completed",
		Message:  "Refund processed successfully",
	}, nil
}
