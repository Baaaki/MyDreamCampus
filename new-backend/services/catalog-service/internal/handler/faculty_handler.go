package handler

import (
	"context"
	"net/http"

	"github.com/baaaki/mydreamcampus/catalog/internal/dto"
	"github.com/baaaki/mydreamcampus/catalog/internal/service"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FacultyHandler struct {
	facultyService *service.FacultyService
}

func NewFacultyHandler(facultyService *service.FacultyService) *FacultyHandler {
	return &FacultyHandler{facultyService: facultyService}
}

// ListFaculties handles GET /api/catalog/faculties
// Role: Public — the course browser filters on it before anyone logs in.
func (h *FacultyHandler) ListFaculties(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	response, err := h.facultyService.ListFaculties(ctx)
	if err != nil {
		reqLogger := logger.WithContextAndFields(ctx,
			zap.String("handler", "FacultyHandler"),
			zap.String("endpoint", "ListFaculties"),
		)
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to list faculties", zap.Error(err), zap.String("error_code", appErr.Code))
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{Error: appErr.Message, Code: appErr.Code})
			return
		}
		reqLogger.Error("unexpected error listing faculties", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
