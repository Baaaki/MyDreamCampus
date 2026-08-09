package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/payment/internal/dto"
	"github.com/baaaki/mydreamcampus/payment/internal/service"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const requestTimeout = 10 * time.Second

// PaymentHandler exposes the mock payment service over internal REST so
// meal can reach it the same way it will once payment runs on its own.
type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: svc}
}

// RegisterInternalRoutes mounts the endpoints meal calls. The caller
// supplies a group already guarded by RequireInternalSecret — payment has
// no user-facing routes at all.
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sharedErrors.ErrInternal.Message,
			"code":  sharedErrors.ErrInternal.Code,
		})
		return
	}

	c.JSON(http.StatusOK, dto.InitiatePaymentResponse{
		PaymentID:  resp.PaymentID,
		PaymentURL: resp.PaymentURL,
		Amount:     resp.Amount,
		Currency:   resp.Currency,
		ExpiresAt:  resp.ExpiresAt,
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sharedErrors.ErrInternal.Message,
			"code":  sharedErrors.ErrInternal.Code,
		})
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
