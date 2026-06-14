package handler

import (
	"context"
	"net/http"
	"time"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/service"
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

	// Create child logger with request context and endpoint info
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "CreateStaff"),
		zap.String("handler", "StaffHandler"),
	)

	var req dto.CreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		reqLogger.Error("invalid request body",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: sharedErrors.ErrValidation.Message,
			Code:  sharedErrors.ErrValidation.Code,
		})
		return
	}

	reqLogger.Info("creating staff",
		zap.String("email", req.Email),
	)

	response, err := h.service.CreateStaff(ctx, req)
	if err != nil {
		// Use modern errors.As for type assertion
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to create staff",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
				zap.String("email", req.Email),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		// Unexpected error (not an AppError)
		reqLogger.Error("unexpected error creating staff",
			zap.Error(err),
			zap.String("email", req.Email),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("staff created successfully",
		zap.String("staff_id", response.ID),
		zap.String("email", req.Email),
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

	// Create child logger with request context and endpoint info
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "GetStaffByID"),
		zap.String("handler", "StaffHandler"),
		zap.String("staff_id", id),
	)

	reqLogger.Info("getting staff by ID")

	response, err := h.service.GetStaffByID(ctx, id)
	if err != nil {
		// Use modern errors.As for type assertion
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Warn("staff not found or error",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		// Unexpected error (not an AppError)
		reqLogger.Error("unexpected error getting staff",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("staff retrieved successfully")
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

	// Create child logger with request context and endpoint info
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "UpdateStaff"),
		zap.String("handler", "StaffHandler"),
		zap.String("staff_id", id),
	)

	var req dto.UpdateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		reqLogger.Error("invalid request body",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: sharedErrors.ErrValidation.Message,
			Code:  sharedErrors.ErrValidation.Code,
		})
		return
	}

	reqLogger.Info("updating staff",
		zap.Any("update_fields", req),
	)

	response, err := h.service.UpdateStaff(ctx, id, req)
	if err != nil {
		// Use modern errors.As for type assertion
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to update staff",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		// Unexpected error (not an AppError)
		reqLogger.Error("unexpected error updating staff",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("staff updated successfully")

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

	// Create child logger with request context and endpoint info
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "DeleteStaff"),
		zap.String("handler", "StaffHandler"),
		zap.String("staff_id", id),
	)

	reqLogger.Info("deleting staff")

	err := h.service.DeleteStaff(ctx, id)
	if err != nil {
		// Use modern errors.As for type assertion
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to delete staff",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		// Unexpected error (not an AppError)
		reqLogger.Error("unexpected error deleting staff",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("staff deleted successfully")

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

	// Create child logger with request context and endpoint info
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "ListStaff"),
		zap.String("handler", "StaffHandler"),
	)

	var query dto.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		reqLogger.Error("invalid query parameters",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: sharedErrors.ErrValidation.Message,
			Code:  sharedErrors.ErrValidation.Code,
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

	reqLogger.Info("listing staff",
		zap.Int("page", query.Page),
		zap.Int("limit", query.Limit),
	)

	response, err := h.service.ListStaff(ctx, query)
	if err != nil {
		// Use modern errors.As for type assertion
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to list staff",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		// Unexpected error (not an AppError)
		reqLogger.Error("unexpected error listing staff",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("staff list retrieved successfully",
		zap.Int("total_results", len(response.Data)),
	)

	c.JSON(http.StatusOK, response)
}

// GetInstructorsByDepartment godoc
// @Summary Get instructors by department
// @Tags staff
// @Produce json
// @Param department query string true "Department name"
// @Success 200 {object} dto.StaffListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /internal/staff/instructors [get]
func (h *StaffHandler) GetInstructorsByDepartment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	department := c.Query("department")
	if department == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Department parameter is required",
			Code:  sharedErrors.ErrValidation.Code,
		})
		return
	}

	// Create child logger with request context and endpoint info
	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "GetInstructorsByDepartment"),
		zap.String("handler", "StaffHandler"),
		zap.String("department", department),
	)

	reqLogger.Info("getting instructors by department")

	response, err := h.service.GetInstructorsByDepartment(ctx, department)
	if err != nil {
		// Use modern errors.As for type assertion
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to get instructors",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		// Unexpected error (not an AppError)
		reqLogger.Error("unexpected error getting instructors",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("instructors retrieved successfully",
		zap.Int("total_results", len(response.Data)),
	)

	c.JSON(http.StatusOK, response)
}
