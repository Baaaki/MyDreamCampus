package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/payment/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/payment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Store is the persistence the payment service needs. The repository
// implements it; tests use an in-memory fake.
type Store interface {
	CreatePayment(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error)
	GetPaymentByReferenceID(ctx context.Context, referenceID string) (db.Payment, error)
	RefundPayment(ctx context.Context, referenceID string, amount float64) (db.Payment, error)
	ExpireOverduePayments(ctx context.Context, now time.Time) (int64, error)
}

// InitiatePaymentRequest is what meal asks to be paid.
type InitiatePaymentRequest struct {
	ReferenceID string
	Amount      float64
	Currency    string
	Description string
	StudentID   string
}

// InitiatePaymentResponse identifies the pending payment the student confirms.
type InitiatePaymentResponse struct {
	PaymentID string
	Amount    float64
	Currency  string
	ExpiresAt time.Time
}

// RefundRequest returns part or all of a completed payment.
type RefundRequest struct {
	ReferenceID string
	Amount      float64
	Currency    string
	Reason      string
}

// RefundResponse reports a processed refund.
type RefundResponse struct {
	RefundID string
	Amount   float64
	Currency string
	Status   string
	Message  string
}

type PaymentService struct {
	store Store
	// timeout matches meal's reservation hold, so the payment and the
	// reservation it pays for lapse together.
	timeout time.Duration
	logger  *zap.Logger
}

func NewPaymentService(store Store, timeout time.Duration, logger *zap.Logger) *PaymentService {
	return &PaymentService{
		store:   store,
		timeout: timeout,
		logger:  logger,
	}
}

// InitiatePayment opens a pending payment for a meal reservation. A retried
// call with the same reference returns the payment it opened the first time.
func (s *PaymentService) InitiatePayment(ctx context.Context, req InitiatePaymentRequest) (*InitiatePaymentResponse, error) {
	studentID, err := uuid.Parse(req.StudentID)
	// Below one kuruş the amount would round to zero in NUMERIC(10,2).
	if err != nil || req.Amount < 0.01 || len(req.Currency) != 3 || !validReference(req.ReferenceID) {
		return nil, serviceErrors.ErrInvalidInitiate
	}

	payment, err := s.store.CreatePayment(ctx, db.CreatePaymentParams{
		ReferenceID: req.ReferenceID,
		StudentID:   utils.UUIDToPgtype(studentID),
		Amount:      utils.Float64ToPgNumeric(req.Amount),
		Currency:    req.Currency,
		Description: req.Description,
		ExpiresAt:   utils.TimeToPgTimestamptz(clock.Now().Add(s.timeout)),
	})
	if err != nil {
		return nil, err
	}

	amount, err := utils.PgNumericToFloat64(payment.Amount)
	if err != nil {
		return nil, err
	}

	s.logger.Info("payment initiated",
		zap.String("payment_id", utils.PgtypeToUUIDString(payment.ID)),
		zap.String("reference_id", payment.ReferenceID),
		zap.String("student_id", studentID.String()),
		zap.String("status", string(payment.Status)),
	)

	return &InitiatePaymentResponse{
		PaymentID: utils.PgtypeToUUIDString(payment.ID),
		Amount:    amount,
		Currency:  payment.Currency,
		ExpiresAt: payment.ExpiresAt.Time,
	}, nil
}

// RequestRefund returns amount from a completed payment. Meal refunds one
// meal at a time, so a batch payment is refunded in parts.
func (s *PaymentService) RequestRefund(ctx context.Context, req RefundRequest) (*RefundResponse, error) {
	if req.Amount < 0.01 {
		return nil, sharedErrors.ErrValidation
	}

	payment, err := s.store.RefundPayment(ctx, req.ReferenceID, req.Amount)
	if errors.Is(err, serviceErrors.ErrPaymentNotFoundRepo) {
		return nil, s.refundRejection(ctx, req.ReferenceID)
	}
	if err != nil {
		return nil, err
	}

	s.logger.Info("payment refunded",
		zap.String("payment_id", utils.PgtypeToUUIDString(payment.ID)),
		zap.String("reference_id", payment.ReferenceID),
		zap.Float64("amount", req.Amount),
		zap.String("status", string(payment.Status)),
	)

	return &RefundResponse{
		// Refunds are recorded on the payment itself, not as rows of their own.
		RefundID: utils.PgtypeToUUIDString(payment.ID),
		Amount:   req.Amount,
		Currency: payment.Currency,
		Status:   "completed",
		Message:  "İade işlendi",
	}, nil
}

// refundRejection explains why no payment qualified for the refund.
func (s *PaymentService) refundRejection(ctx context.Context, referenceID string) error {
	payment, err := s.store.GetPaymentByReferenceID(ctx, referenceID)
	if errors.Is(err, serviceErrors.ErrPaymentNotFoundRepo) {
		return serviceErrors.ErrPaymentNotFound
	}
	if err != nil {
		return err
	}
	if payment.Status != db.PaymentPaymentStatusEnumCompleted {
		return serviceErrors.ErrRefundNotAllowed
	}
	return serviceErrors.ErrRefundTooLarge
}

// ExpireOverdue marks pending payments past their deadline as expired. No
// event follows: meal drops the reservation on its own timer.
func (s *PaymentService) ExpireOverdue(ctx context.Context) (int64, error) {
	return s.store.ExpireOverduePayments(ctx, clock.Now())
}

// validReference accepts the two shapes meal's payment consumer can route:
// "res_<uuid>" for one reservation and "bat_<uuid>" for a batch.
func validReference(ref string) bool {
	rest, ok := strings.CutPrefix(ref, "res_")
	if !ok {
		rest, ok = strings.CutPrefix(ref, "bat_")
	}
	if !ok {
		return false
	}
	_, err := uuid.Parse(rest)
	return err == nil
}
