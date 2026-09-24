package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/payment/internal/card"
	"github.com/baaaki/mydreamcampus/payment/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/payment/internal/errors"
	"github.com/baaaki/mydreamcampus/payment/internal/service"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const requestTimeout = 10 * time.Second

// PaymentHandler exposes the payment service to meal over internal REST and
// to the student who confirms the payment over /api/payments.
type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: svc}
}

// RegisterRoutes mounts the student's checkout endpoints. The caller
// supplies a group that already authenticates and requires a student.
// Confirm charges the card, so a retried request must not charge twice.
func (h *PaymentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/:payment_id", h.GetPayment)
	rg.POST("/:payment_id/confirm", platformMiddleware.Idempotency(), h.ConfirmPayment)
}

// ConfirmPayment handles POST /api/payments/:payment_id/confirm.
func (h *PaymentHandler) ConfirmPayment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "PaymentHandler"),
		zap.String("method", "ConfirmPayment"),
	)

	studentID, paymentID, ok := ids(c)
	if !ok {
		return
	}

	var req dto.ConfirmPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// The decoder error is not logged: its message can quote the input,
		// and the input is a card number.
		reqLogger.Warn("invalid confirm body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": sharedErrors.ErrValidation.Message,
			"code":  sharedErrors.ErrValidation.Code,
		})
		return
	}

	view, err := h.service.ConfirmPayment(ctx, studentID, paymentID, card.Input{
		Number:   req.CardNumber,
		ExpMonth: req.ExpMonth,
		ExpYear:  req.ExpYear,
		CVC:      req.CVC,
		Holder:   req.CardholderName,
	})
	if err != nil {
		logServiceError(reqLogger, "payment confirmation failed", err, paymentID)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPaymentResponse(view))
}

// GetPayment handles GET /api/payments/:payment_id.
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	studentID, paymentID, ok := ids(c)
	if !ok {
		return
	}

	view, err := h.service.GetPayment(ctx, studentID, paymentID)
	if err != nil {
		reqLogger := logger.WithContextAndFields(ctx,
			zap.String("handler", "PaymentHandler"),
			zap.String("method", "GetPayment"),
		)
		logServiceError(reqLogger, "payment lookup failed", err, paymentID)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPaymentResponse(view))
}

// ids reads the caller from the token and the payment from the path. A
// malformed payment id is answered like an unknown one.
func ids(c *gin.Context) (studentID, paymentID uuid.UUID, ok bool) {
	studentID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		writeError(c, sharedErrors.ErrUnauthorized)
		return uuid.Nil, uuid.Nil, false
	}
	paymentID, err = uuid.Parse(c.Param("payment_id"))
	if err != nil {
		writeError(c, serviceErrors.ErrPaymentNotFound)
		return uuid.Nil, uuid.Nil, false
	}
	return studentID, paymentID, true
}

// logServiceError keeps expected rejections (a mistyped card, an expired
// payment) at warn so they do not read as incidents.
func logServiceError(log *zap.Logger, msg string, err error, paymentID uuid.UUID) {
	var appErr *sharedErrors.AppError
	if errors.As(err, &appErr) && appErr.HTTPStatus < http.StatusInternalServerError {
		log.Warn(msg, zap.String("code", appErr.Code), zap.String("payment_id", paymentID.String()))
		return
	}
	log.Error(msg, zap.Error(err), zap.String("payment_id", paymentID.String()))
}

func toPaymentResponse(v *service.PaymentView) dto.PaymentResponse {
	return dto.PaymentResponse{
		ID:            v.ID,
		Status:        v.Status,
		Amount:        v.Amount,
		Currency:      v.Currency,
		CardBrand:     v.CardBrand,
		CardLast4:     v.CardLast4,
		FailureReason: v.FailureReason,
		ExpiresAt:     v.ExpiresAt,
		CompletedAt:   v.CompletedAt,
	}
}

// RegisterInternalRoutes mounts the endpoints meal calls. The caller
// supplies a group already guarded by RequireInternalSecret.
func (h *PaymentHandler) RegisterInternalRoutes(rg *gin.RouterGroup) {
	rg.POST("/payments/initiate", h.InitiatePayment)
	rg.POST("/payments/refund", h.RequestRefund)
}

// InitiatePayment handles POST /internal/payments/initiate.
func (h *PaymentHandler) InitiatePayment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "PaymentHandler"),
		zap.String("method", "InitiatePayment"),
	)

	var req dto.InitiatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		reqLogger.Warn("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": sharedErrors.ErrValidation.Message,
			"code":  sharedErrors.ErrValidation.Code,
		})
		return
	}

	resp, err := h.service.InitiatePayment(ctx, service.InitiatePaymentRequest{
		ReferenceID: req.ReferenceID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		StudentID:   req.StudentID,
	})
	if err != nil {
		reqLogger.Error("payment initiation failed", zap.Error(err),
			zap.String("reference_id", req.ReferenceID))
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.InitiatePaymentResponse{
		PaymentID: resp.PaymentID,
		Amount:    resp.Amount,
		Currency:  resp.Currency,
		ExpiresAt: resp.ExpiresAt.Format(time.RFC3339),
	})
}

// RequestRefund handles POST /internal/payments/refund.
func (h *PaymentHandler) RequestRefund(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "PaymentHandler"),
		zap.String("method", "RequestRefund"),
	)

	var req dto.RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		reqLogger.Warn("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": sharedErrors.ErrValidation.Message,
			"code":  sharedErrors.ErrValidation.Code,
		})
		return
	}

	resp, err := h.service.RequestRefund(ctx, service.RefundRequest{
		ReferenceID: req.ReferenceID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Reason:      req.Reason,
	})
	if err != nil {
		reqLogger.Error("refund failed", zap.Error(err),
			zap.String("reference_id", req.ReferenceID))
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.RefundResponse{
		RefundID: resp.RefundID,
		Amount:   resp.Amount,
		Currency: resp.Currency,
		Status:   resp.Status,
		Message:  resp.Message,
	})
}

// writeError answers with the AppError's own status and message, and hides
// anything else behind a generic 500.
func writeError(c *gin.Context, err error) {
	var appErr *sharedErrors.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": sharedErrors.ErrInternal.Message,
		"code":  sharedErrors.ErrInternal.Code,
	})
}
