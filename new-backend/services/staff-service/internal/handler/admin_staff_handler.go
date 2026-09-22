package handler

import (
	"context"
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/staff/internal/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminStaffService is what AdminStaffHandler calls;
// service.AdminStaffService implements it.
type AdminStaffService interface {
	List(ctx context.Context, faculty string) (dto.AdminStaffListResponse, error)
	Get(ctx context.Context, id string) (dto.AdminStaffResponse, error)
	Create(ctx context.Context, req dto.AdminStaffRequest) (dto.AdminStaffResponse, error)
	Update(ctx context.Context, id string, req dto.AdminStaffRequest) (dto.AdminStaffResponse, error)
}

// AdminStaffHandler serves /api/admin-staff. Every route is admin-only; the
// module wires the guard.
type AdminStaffHandler struct {
	service AdminStaffService
}

func NewAdminStaffHandler(service AdminStaffService) *AdminStaffHandler {
	return &AdminStaffHandler{service: service}
}

// List godoc
// @Summary List administrative staff
// @Tags admin-staff
// @Produce json
// @Param faculty query string false "Faculty name"
// @Success 200 {object} dto.AdminStaffListResponse
// @Router /api/admin-staff [get]
func (h *AdminStaffHandler) List(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	resp, err := h.service.List(ctx, c.Query("faculty"))
	if err != nil {
		writeAdminStaffError(ctx, c, "List", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Get godoc
// @Summary Get one administrative staff record
// @Tags admin-staff
// @Produce json
// @Param id path string true "Admin staff ID"
// @Success 200 {object} dto.AdminStaffResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/admin-staff/{id} [get]
func (h *AdminStaffHandler) Get(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	resp, err := h.service.Get(ctx, c.Param("id"))
	if err != nil {
		writeAdminStaffError(ctx, c, "Get", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Create godoc
// @Summary Create an administrative staff record
// @Tags admin-staff
// @Accept json
// @Produce json
// @Param request body dto.AdminStaffRequest true "Record"
// @Success 201 {object} dto.AdminStaffResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /api/admin-staff [post]
func (h *AdminStaffHandler) Create(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req dto.AdminStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithContext(ctx).Warn("invalid admin staff body", zap.Error(err))
		writeAdminStaffError(ctx, c, "Create", sharedErrors.ErrValidation)
		return
	}
	resp, err := h.service.Create(ctx, req)
	if err != nil {
		writeAdminStaffError(ctx, c, "Create", err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// Update godoc
// @Summary Replace an administrative staff record
// @Tags admin-staff
// @Accept json
// @Produce json
// @Param id path string true "Admin staff ID"
// @Param request body dto.AdminStaffRequest true "Record"
// @Success 200 {object} dto.AdminStaffResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Router /api/admin-staff/{id} [put]
func (h *AdminStaffHandler) Update(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req dto.AdminStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithContext(ctx).Warn("invalid admin staff body", zap.Error(err))
		writeAdminStaffError(ctx, c, "Update", sharedErrors.ErrValidation)
		return
	}
	resp, err := h.service.Update(ctx, c.Param("id"), req)
	if err != nil {
		writeAdminStaffError(ctx, c, "Update", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// writeAdminStaffError answers with the AppError's status and Turkish text;
// anything else is a 500 whose detail stays in the log.
func writeAdminStaffError(ctx context.Context, c *gin.Context, endpoint string, err error) {
	log := logger.WithContextAndFields(ctx,
		zap.String("handler", "AdminStaffHandler"),
		zap.String("endpoint", endpoint),
	)
	appErr, ok := sharedErrors.As(err)
	if !ok || appErr.HTTPStatus >= http.StatusInternalServerError {
		log.Error("admin staff request failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}
	log.Warn("admin staff request rejected", zap.Error(err), zap.String("error_code", appErr.Code))
	c.JSON(appErr.HTTPStatus, dto.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
}
