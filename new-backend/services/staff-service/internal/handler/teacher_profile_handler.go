package handler

import (
	"context"
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/staff/internal/dto"
	"github.com/baaaki/mydreamcampus/staff/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TeacherProfileHandler struct {
	service *service.TeacherProfileService
}

func NewTeacherProfileHandler(service *service.TeacherProfileService) *TeacherProfileHandler {
	return &TeacherProfileHandler{
		service: service,
	}
}

// GetTeacherProfileByStaffID godoc
// @Summary Get teacher profile by staff ID (public)
// @Description Retrieves public academic profile for a teacher
// @Tags teacher-profiles
// @Produce json
// @Param id path string true "Staff ID"
// @Success 200 {object} dto.TeacherProfileResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/staff/profile/{id} [get]
func (h *TeacherProfileHandler) GetTeacherProfileByStaffID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	staffID := c.Param("id")

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "GetTeacherProfileByStaffID"),
		zap.String("handler", "TeacherProfileHandler"),
		zap.String("staff_id", staffID),
	)

	reqLogger.Info("getting teacher profile by staff ID")

	response, err := h.service.GetTeacherProfileByStaffID(ctx, staffID)
	if err != nil {
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Warn("failed to get teacher profile",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error getting teacher profile", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("teacher profile retrieved successfully")
	c.JSON(http.StatusOK, response)
}

// UpdateTeacherProfile godoc
// @Summary Update teacher profile
// @Description Updates academic profile for a teacher (admin only)
// @Tags teacher-profiles
// @Accept json
// @Produce json
// @Param id path string true "Staff ID"
// @Param request body dto.UpdateTeacherProfileRequest true "Update teacher profile request"
// @Success 200 {object} dto.TeacherProfileResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/staff/{id}/profile [put]
func (h *TeacherProfileHandler) UpdateTeacherProfile(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	staffID := c.Param("id")

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("endpoint", "UpdateTeacherProfile"),
		zap.String("handler", "TeacherProfileHandler"),
		zap.String("staff_id", staffID),
	)

	var req dto.UpdateTeacherProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		reqLogger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: sharedErrors.ErrValidation.Message,
			Code:  sharedErrors.ErrValidation.Code,
		})
		return
	}

	reqLogger.Info("updating teacher profile")

	response, err := h.service.UpdateTeacherProfile(ctx, staffID, req)
	if err != nil {
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to update teacher profile",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error updating teacher profile", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("teacher profile updated successfully")
	c.JSON(http.StatusOK, response)
}
