package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/staff-service/internal/dto"
	"github.com/baaaki/mydreamcampus/staff-service/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	requestTimeout = 5 * time.Second
	maxPageLimit   = 100
	defaultLimit   = 20
)

type StaffHandler struct {
	service *service.StaffService
}

func NewStaffHandler(service *service.StaffService) *StaffHandler {
	return &StaffHandler{
		service: service,
	}
}

// CreateStaff godoc
// @Summary Create a new staff member
// @Tags staff
// @Accept json
// @Produce json
// @Param request body dto.CreateStaffRequest true "Create staff request"
// @Success 201 {object} dto.StaffResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /staff [post]
func (h *StaffHandler) CreateStaff(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var req dto.CreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body",
			zap.Error(err),
			zap.String("endpoint", "CreateStaff"),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: errors.ErrValidation.Message,
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	logger.Info("creating staff",
		zap.String("email", req.Email),
	)

	response, err := h.service.CreateStaff(ctx, req)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	logger.Info("staff created successfully",
		zap.String("staff_id", response.ID),
	)

	c.JSON(http.StatusCreated, response)
}

// GetStaffByID godoc
// @Summary Get staff by ID
// @Tags staff
// @Produce json
// @Param id path string true "Staff ID"
// @Success 200 {object} dto.StaffResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /staff/{id} [get]
func (h *StaffHandler) GetStaffByID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	id := c.Param("id")

	logger.Info("getting staff by ID",
		zap.String("staff_id", id),
	)

	response, err := h.service.GetStaffByID(ctx, id)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateStaff godoc
// @Summary Update staff information
// @Tags staff
// @Accept json
// @Produce json
// @Param id path string true "Staff ID"
// @Param request body dto.UpdateStaffRequest true "Update staff request"
// @Success 200 {object} dto.StaffResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /staff/{id} [put]
func (h *StaffHandler) UpdateStaff(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	id := c.Param("id")

	var req dto.UpdateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body",
			zap.Error(err),
			zap.String("endpoint", "UpdateStaff"),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: errors.ErrValidation.Message,
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	logger.Info("updating staff",
		zap.String("staff_id", id),
	)

	response, err := h.service.UpdateStaff(ctx, id, req)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	logger.Info("staff updated successfully",
		zap.String("staff_id", id),
	)

	c.JSON(http.StatusOK, response)
}

// DeleteStaff godoc
// @Summary Delete staff (soft delete)
// @Tags staff
// @Produce json
// @Param id path string true "Staff ID"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /staff/{id} [delete]
func (h *StaffHandler) DeleteStaff(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	id := c.Param("id")

	logger.Info("deleting staff",
		zap.String("staff_id", id),
	)

	err := h.service.DeleteStaff(ctx, id)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	logger.Info("staff deleted successfully",
		zap.String("staff_id", id),
	)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Staff deleted successfully",
	})
}

// ListStaff godoc
// @Summary List staff with pagination
// @Tags staff
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} dto.StaffListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /staff [get]
func (h *StaffHandler) ListStaff(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	var query dto.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Error("invalid query parameters",
			zap.Error(err),
			zap.String("endpoint", "ListStaff"),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: errors.ErrValidation.Message,
			Code:  errors.ErrValidation.Code,
		})
		return
	}

	// Set defaults and apply limits
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxPageLimit {
		query.Limit = maxPageLimit
	}
	if query.Limit < 1 {
		query.Limit = defaultLimit
	}

	logger.Info("listing staff",
		zap.Int("page", query.Page),
		zap.Int("limit", query.Limit),
	)

	response, err := h.service.ListStaff(ctx, query)
	if err != nil {
		appErr, ok := err.(*errors.AppError)
		if ok {
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: errors.ErrInternal.Message,
			Code:  errors.ErrInternal.Code,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
