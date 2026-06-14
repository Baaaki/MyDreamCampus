package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/baaaki/mydreamcampus/meal-service/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/meal-service/internal/errors"
	"github.com/baaaki/mydreamcampus/meal-service/internal/service"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MealHandler struct {
	cafeteriaService   *service.CafeteriaService
	reservationService *service.ReservationService
	menuService        *service.MenuService
	logger             *zap.Logger
}

func NewMealHandler(
	cafeteriaService *service.CafeteriaService,
	reservationService *service.ReservationService,
	menuService *service.MenuService,
	logger *zap.Logger,
) *MealHandler {
	return &MealHandler{
		cafeteriaService:   cafeteriaService,
		reservationService: reservationService,
		menuService:        menuService,
		logger:             logger,
	}
}

// ============================================================================
// CAFETERIA ENDPOINTS
// ============================================================================

// GetCafeterias godoc
// @Summary Get cafeterias (admin sees all, others see only active)
// @Tags cafeterias
// @Accept json
// @Produce json
// @Success 200 {object} dto.CafeteriaListResponse
// @Router /api/v1/meals/cafeterias [get]
func (h *MealHandler) GetCafeterias(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Check if user is admin to show all cafeterias including inactive
	role, _ := c.Get("role")

	var cafeterias *dto.CafeteriaListResponse
	var err error

	if role == "admin" {
		cafeterias, err = h.cafeteriaService.GetAllCafeterias(ctx)
	} else {
		cafeterias, err = h.cafeteriaService.GetActiveCafeterias(ctx)
	}

	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    cafeterias,
	})
}

// CreateCafeteria godoc
// @Summary Create cafeteria (Admin only)
// @Tags cafeterias
// @Accept json
// @Produce json
// @Param request body dto.CreateCafeteriaRequest true "Cafeteria data"
// @Success 201 {object} dto.CafeteriaResponse
// @Router /api/v1/meals/cafeterias [post]
func (h *MealHandler) CreateCafeteria(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req dto.CreateCafeteriaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", zap.Error(err))
		h.handleError(c, sharedErrors.ErrValidation)
		return
	}

	cafeteria, err := h.cafeteriaService.CreateCafeteria(ctx, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    cafeteria,
	})
}

// UpdateCafeteria godoc
// @Summary Update cafeteria (Admin only)
// @Tags cafeterias
// @Accept json
// @Produce json
// @Param cafeteria_id path string true "Cafeteria ID"
// @Param request body dto.UpdateCafeteriaRequest true "Cafeteria data"
// @Success 200 {object} dto.CafeteriaResponse
// @Router /api/v1/meals/cafeterias/{cafeteria_id} [put]
func (h *MealHandler) UpdateCafeteria(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	cafeteriaID := c.Param("cafeteria_id")

	var req dto.UpdateCafeteriaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", zap.Error(err))
		h.handleError(c, sharedErrors.ErrValidation)
		return
	}

	cafeteria, err := h.cafeteriaService.UpdateCafeteria(ctx, cafeteriaID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    cafeteria,
	})
}

// DeleteCafeteria godoc
// @Summary Delete cafeteria (Admin only)
// @Tags cafeterias
// @Accept json
// @Produce json
// @Param cafeteria_id path string true "Cafeteria ID"
// @Success 200 {object} dto.MessageResponse
// @Router /api/v1/meals/cafeterias/{cafeteria_id} [delete]
func (h *MealHandler) DeleteCafeteria(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	cafeteriaID := c.Param("cafeteria_id")

	err := h.cafeteriaService.DeactivateCafeteria(ctx, cafeteriaID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data: dto.MessageResponse{
			Message: "Cafeteria deactivated",
			ID:      cafeteriaID,
		},
	})
}

// ============================================================================
// RESERVATION ENDPOINTS
// ============================================================================

// CreateReservation godoc
// @Summary Create single reservation (Student only)
// @Tags reservations
// @Accept json
// @Produce json
// @Param request body dto.CreateReservationRequest true "Reservation data"
// @Success 200 {object} dto.CreateReservationResponse
// @Router /api/v1/meals/reservations [post]
func (h *MealHandler) CreateReservation(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Get student ID from JWT
	studentID, err := h.getStudentIDFromContext(c)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var req dto.CreateReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", zap.Error(err))
		h.handleError(c, sharedErrors.ErrValidation)
		return
	}

	reservation, err := h.reservationService.CreateReservation(ctx, studentID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    reservation,
	})
}

// CreateBatchReservation godoc
// @Summary Create batch reservations (Student only)
// @Tags reservations
// @Accept json
// @Produce json
// @Param request body dto.BatchReservationRequest true "Batch reservation data"
// @Success 200 {object} dto.CreateBatchReservationResponse
// @Router /api/v1/meals/reservations/batch [post]
func (h *MealHandler) CreateBatchReservation(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Get student ID from JWT
	studentID, err := h.getStudentIDFromContext(c)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var req dto.BatchReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", zap.Error(err))
		h.handleError(c, sharedErrors.ErrValidation)
		return
	}

	reservation, err := h.reservationService.CreateBatchReservation(ctx, studentID, req)
	if err != nil {
		h.logger.Error("failed to create batch reservation", zap.Error(err))
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    reservation,
	})
}

// GetMyReservations godoc
// @Summary Get my reservations (Student only)
// @Tags reservations
// @Accept json
// @Produce json
// @Param from_date query string false "From date (YYYY-MM-DD)"
// @Param to_date query string false "To date (YYYY-MM-DD)"
// @Param status query string false "Status filter"
// @Success 200 {object} dto.MyReservationsResponse
// @Router /api/v1/meals/reservations/my [get]
func (h *MealHandler) GetMyReservations(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Get student ID from JWT
	studentID, err := h.getStudentIDFromContext(c)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var query dto.MyReservationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Error("invalid query parameters", zap.Error(err))
		h.handleError(c, sharedErrors.ErrValidation)
		return
	}

	reservations, err := h.reservationService.GetMyReservations(ctx, studentID, query)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    reservations,
	})
}

// CancelReservation godoc
// @Summary Cancel reservation (Student only)
// @Tags reservations
// @Accept json
// @Produce json
// @Param reservation_id path string true "Reservation ID"
// @Success 200 {object} dto.CancelReservationResponse
// @Router /api/v1/meals/reservations/{reservation_id} [delete]
func (h *MealHandler) CancelReservation(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Get student ID from JWT
	studentID, err := h.getStudentIDFromContext(c)
	if err != nil {
		h.handleError(c, err)
		return
	}

	reservationID := c.Param("reservation_id")

	response, err := h.reservationService.CancelReservation(ctx, studentID, reservationID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    response,
	})
}

// UseReservation godoc
// @Summary Use reservation with QR (Student only)
// @Tags reservations
// @Accept json
// @Produce json
// @Param request body dto.UseReservationRequest true "QR payload"
// @Success 200 {object} dto.UseReservationResponse
// @Router /api/v1/meals/reservations/use [post]
func (h *MealHandler) UseReservation(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	// Get student ID from JWT
	studentID, err := h.getStudentIDFromContext(c)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var req dto.UseReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", zap.Error(err))
		h.handleError(c, sharedErrors.ErrValidation)
		return
	}

	response, err := h.reservationService.UseReservation(ctx, studentID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    response,
	})
}

// ============================================================================
// MENU ENDPOINTS
// ============================================================================

// CreateMonthlyMenu godoc
// @Summary Create/update monthly menu (Admin only)
// @Tags menus
// @Accept json
// @Produce json
// @Param request body dto.CreateMonthlyMenuRequest true "Menu data"
// @Success 201 {object} dto.MonthlyMenuResponse
// @Router /api/v1/meals/menu/monthly [post]
func (h *MealHandler) CreateMonthlyMenu(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req dto.CreateMonthlyMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", zap.Error(err))
		h.handleError(c, sharedErrors.ErrValidation)
		return
	}

	menu, err := h.menuService.CreateOrUpdateMonthlyMenu(ctx, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Data:    menu,
	})
}

// GetMonthlyMenu godoc
// @Summary Get monthly menu (Public)
// @Tags menus
// @Accept json
// @Produce json
// @Param year query int false "Year"
// @Param month query int false "Month"
// @Success 200 {object} dto.MonthlyMenuResponse
// @Router /api/v1/meals/menu/monthly [get]
func (h *MealHandler) GetMonthlyMenu(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var query dto.GetMonthlyMenuQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Error("invalid query parameters", zap.Error(err))
		h.handleError(c, sharedErrors.ErrValidation)
		return
	}

	menu, err := h.menuService.GetMonthlyMenu(ctx, query.Year, query.Month)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    menu,
	})
}

// ============================================================================
// QR GENERATION ENDPOINT
// ============================================================================

// GenerateQR godoc
// @Summary Generate QR code for cafeteria (Admin only)
// @Tags qr
// @Accept json
// @Produce json
// @Param cafeteria_id path string true "Cafeteria ID"
// @Param date query string false "Date (YYYY-MM-DD)"
// @Param meal_time query string true "Meal time (lunch/dinner)"
// @Success 200 {object} dto.QRResponse
// @Router /api/v1/meals/cafeterias/{cafeteria_id}/qr [get]
func (h *MealHandler) GenerateQR(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	cafeteriaID := c.Param("cafeteria_id")

	var query dto.GenerateQRRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Error("invalid query parameters", zap.Error(err))
		h.handleError(c, sharedErrors.ErrValidation)
		return
	}

	qr, err := h.reservationService.GenerateQR(ctx, cafeteriaID, query.Date, query.MealTime)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    qr,
	})
}

// ============================================================================
// HEALTH ENDPOINTS
// ============================================================================

// Health godoc
// @Summary Health check
// @Tags health
// @Accept json
// @Produce json
// @Success 200
// @Router /health [get]
func (h *MealHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "meal-service",
	})
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (h *MealHandler) getStudentIDFromContext(c *gin.Context) (uuid.UUID, error) {
	// Role check is enforced by middleware.RequireStudent(); we only parse the ID here.
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.UUID{}, sharedErrors.ErrUnauthorized
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return uuid.UUID{}, sharedErrors.ErrUnauthorized
	}

	return userID, nil
}

func (h *MealHandler) handleError(c *gin.Context, err error) {
	// Log error
	h.logger.Error("handler error", zap.Error(err))

	// Map error to HTTP response
	var appErr *sharedErrors.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, dto.ErrorResponseWrapper{
			Success: false,
			Error: dto.ErrorResponse{
				Code:    appErr.Code,
				Message: appErr.Message,
			},
		})
		return
	}

	// Handle custom service errors
	switch {
	case errors.Is(err, serviceErrors.ErrValidationErrors):
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWrapper{
			Success: false,
			Error: dto.ErrorResponse{
				Code:    "VALIDATION_ERRORS",
				Message: "Some reservations have validation errors",
			},
		})
	case errors.Is(err, serviceErrors.ErrReservationConflicts):
		c.JSON(http.StatusConflict, dto.ErrorResponseWrapper{
			Success: false,
			Error: dto.ErrorResponse{
				Code:    "RESERVATION_CONFLICTS",
				Message: "Some dates/meals already have active reservations",
			},
		})
	default:
		// Unknown error
		c.JSON(http.StatusInternalServerError, dto.ErrorResponseWrapper{
			Success: false,
			Error: dto.ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "An unexpected error occurred",
			},
		})
	}
}
