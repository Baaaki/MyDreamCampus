package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/dto"
	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/service"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	requestTimeout = 10 * time.Second
	defaultLimit   = 20
	maxPageLimit   = 100
)

type CatalogHandler struct {
	catalogService *service.CatalogService
}

func NewCatalogHandler(catalogService *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{
		catalogService: catalogService,
	}
}

// CreateCourse handles POST /api/v1/catalog/courses
// Role: Admin
func (h *CatalogHandler) CreateCourse(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "CatalogHandler"),
		zap.String("endpoint", "CreateCourse"),
	)

	var req dto.CreateCourseRequest
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

	reqLogger.Info("creating course in catalog",
		zap.String("course_code", req.CourseCode),
		zap.String("name", req.Name),
		zap.Int16("class_level", req.ClassLevel),
	)

	response, err := h.catalogService.CreateCourse(ctx, req)
	if err != nil {
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to create course",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error creating course",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("course created successfully in catalog",
		zap.String("course_id", response.ID.String()),
		zap.String("course_code", response.CourseCode),
	)

	c.JSON(http.StatusCreated, response)
}

// GetCourseByCourseCode handles GET /api/v1/catalog/courses/:course_code
// Role: Authenticated
func (h *CatalogHandler) GetCourseByCourseCode(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	courseCode := c.Param("course_code")

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "CatalogHandler"),
		zap.String("endpoint", "GetCourseByCourseCode"),
		zap.String("course_code", courseCode),
	)

	reqLogger.Info("fetching course from catalog")

	response, err := h.catalogService.GetCourseByCourseCode(ctx, courseCode)
	if err != nil {
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Warn("course not found or error",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error fetching course",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("course fetched successfully from catalog",
		zap.String("course_id", response.ID.String()),
	)

	c.JSON(http.StatusOK, response)
}

// ListCourses handles GET /api/v1/catalog/courses
// Role: Authenticated
func (h *CatalogHandler) ListCourses(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "CatalogHandler"),
		zap.String("endpoint", "ListCourses"),
	)

	var req dto.ListCoursesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		reqLogger.Error("invalid query parameters",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: sharedErrors.ErrValidation.Message,
			Code:  sharedErrors.ErrValidation.Code,
		})
		return
	}

	// Set default pagination
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = defaultLimit
	}
	if req.Limit > maxPageLimit {
		req.Limit = maxPageLimit
	}

	reqLogger.Info("listing courses from catalog",
		zap.Int("page", req.Page),
		zap.Int("limit", req.Limit),
	)

	response, err := h.catalogService.ListCourses(ctx, req)
	if err != nil {
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to list courses",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error listing courses",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("courses listed successfully from catalog",
		zap.Int("total_records", response.Pagination.Total),
		zap.Int("returned_records", len(response.Data)),
	)

	c.JSON(http.StatusOK, response)
}

// UpdateCourse handles PUT /api/v1/catalog/courses/:course_code
// Role: Admin
func (h *CatalogHandler) UpdateCourse(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	courseCode := c.Param("course_code")

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "CatalogHandler"),
		zap.String("endpoint", "UpdateCourse"),
		zap.String("course_code", courseCode),
	)

	var req dto.UpdateCourseRequest
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

	reqLogger.Info("updating course in catalog")

	response, err := h.catalogService.UpdateCourse(ctx, courseCode, req)
	if err != nil {
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to update course",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error updating course",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("course updated successfully in catalog",
		zap.String("course_id", response.ID.String()),
	)

	c.JSON(http.StatusOK, response)
}
