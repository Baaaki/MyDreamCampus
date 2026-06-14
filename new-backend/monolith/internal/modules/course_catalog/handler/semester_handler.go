package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/dto"
	catalogErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SemesterHandler struct {
	semesterService *service.SemesterService
}

func NewSemesterHandler(semesterService *service.SemesterService) *SemesterHandler {
	return &SemesterHandler{
		semesterService: semesterService,
	}
}

// CreateSemesterCourse handles POST /api/v1/semesters/:semester_id/courses
// Role: Admin
func (h *SemesterHandler) CreateSemesterCourse(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	semester := c.Param("semester_id")

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterHandler"),
		zap.String("endpoint", "CreateSemesterCourse"),
		zap.String("semester", semester),
	)

	var req dto.CreateSemesterCourseRequest
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

	reqLogger.Info("creating semester course",
		zap.String("course_code", req.CourseCode),
		zap.Int16("class_level", req.ClassLevel),
		zap.String("instructor_id", req.InstructorID.String()),
	)

	response, err := h.semesterService.CreateSemesterCourse(ctx, semester, req)
	if err != nil {
		// Check for cross-department schedule conflict (carries details)
		var conflictErr *catalogErrors.ScheduleConflictError
		if errors.As(err, &conflictErr) {
			reqLogger.Error("instructor cross-department schedule conflict",
				zap.Error(err),
				zap.String("conflicting_course", conflictErr.CourseCode),
				zap.String("conflicting_department", conflictErr.Department),
			)
			c.JSON(conflictErr.AppErr.HTTPStatus, dto.ErrorResponse{
				Error: conflictErr.AppErr.Message,
				Code:  conflictErr.AppErr.Code,
				Details: map[string]any{
					"course_code": conflictErr.CourseCode,
					"department":  conflictErr.Department,
					"day_of_week": conflictErr.DayOfWeek,
					"slot_number": conflictErr.SlotNumber,
				},
			})
			return
		}

		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to create semester course",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error creating semester course",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("semester course created successfully",
		zap.String("semester_course_id", response.ID.String()),
		zap.String("course_code", response.CourseCode),
	)

	c.JSON(http.StatusCreated, response)
}

// GetSemesterCourseByID handles GET /api/v1/semesters/:semester_id/courses/:course_id
// Role: Authenticated
func (h *SemesterHandler) GetSemesterCourseByID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	semester := c.Param("semester_id")
	courseID := c.Param("course_id")

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterHandler"),
		zap.String("endpoint", "GetSemesterCourseByID"),
		zap.String("semester", semester),
		zap.String("course_id", courseID),
	)

	reqLogger.Info("fetching semester course")

	response, err := h.semesterService.GetSemesterCourseByID(ctx, semester, courseID)
	if err != nil {
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Warn("semester course not found or error",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error fetching semester course",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("semester course fetched successfully",
		zap.String("course_code", response.CourseCode),
	)

	c.JSON(http.StatusOK, response)
}

// ListSemesterCourses handles GET /api/v1/semesters/:semester_id/courses
// Role: Authenticated
func (h *SemesterHandler) ListSemesterCourses(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	semester := c.Param("semester_id")

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterHandler"),
		zap.String("endpoint", "ListSemesterCourses"),
		zap.String("semester", semester),
	)

	var req dto.ListSemesterCoursesRequest
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

	reqLogger.Info("listing semester courses",
		zap.Int("page", req.Page),
		zap.Int("limit", req.Limit),
	)

	response, err := h.semesterService.ListSemesterCourses(ctx, semester, req)
	if err != nil {
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to list semester courses",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error listing semester courses",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("semester courses listed successfully",
		zap.Int("total_records", response.Pagination.Total),
		zap.Int("returned_records", len(response.Data)),
	)

	c.JSON(http.StatusOK, response)
}

// GetTeacherCourses handles GET /api/v1/semesters/teacher/courses
// Role: Teacher
func (h *SemesterHandler) GetTeacherCourses(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterHandler"),
		zap.String("endpoint", "GetTeacherCourses"),
	)

	// Get instructor ID from context (set by auth middleware)
	instructorIDRaw, exists := c.Get("user_id")
	if !exists {
		reqLogger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	// Handle both string and uuid.UUID types (middleware may set either)
	var instructorID uuid.UUID
	var parseErr error
	switch v := instructorIDRaw.(type) {
	case string:
		instructorID, parseErr = uuid.Parse(v)
		if parseErr != nil {
			reqLogger.Error("failed to parse user_id string as UUID",
				zap.String("user_id_raw", v),
				zap.Error(parseErr),
			)
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error: "invalid user_id format",
				Code:  "INVALID_USER_ID",
			})
			return
		}
	case uuid.UUID:
		instructorID = v
	default:
		reqLogger.Error("invalid user_id type in context")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid user_id",
			Code:  "INVALID_USER_ID",
		})
		return
	}

	// Get optional semester filter
	var semester *string
	if s := c.Query("semester"); s != "" {
		semester = &s
	}

	reqLogger.Info("getting teacher courses",
		zap.String("instructor_id", instructorID.String()),
	)

	response, err := h.semesterService.GetTeacherCourses(ctx, instructorID, semester)
	if err != nil {
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to get teacher courses",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error getting teacher courses",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("teacher courses retrieved successfully",
		zap.Int("total_courses", response.TotalCourses),
	)

	c.JSON(http.StatusOK, response)
}

// DeleteSemesterCourse handles DELETE /api/v1/semesters/:semester_id/courses/:course_id
// Role: Admin
func (h *SemesterHandler) DeleteSemesterCourse(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	semester := c.Param("semester_id")
	courseID := c.Param("course_id")

	reqLogger := logger.WithContextAndFields(ctx,
		zap.String("handler", "SemesterHandler"),
		zap.String("endpoint", "DeleteSemesterCourse"),
		zap.String("semester", semester),
		zap.String("course_id", courseID),
	)

	reqLogger.Info("deleting semester course")

	response, err := h.semesterService.DeleteSemesterCourse(ctx, semester, courseID)
	if err != nil {
		if appErr, ok := sharedErrors.As(err); ok {
			reqLogger.Error("failed to delete semester course",
				zap.Error(err),
				zap.String("error_code", appErr.Code),
			)
			c.JSON(appErr.HTTPStatus, dto.ErrorResponse{
				Error: appErr.Message,
				Code:  appErr.Code,
			})
			return
		}

		reqLogger.Error("unexpected error deleting semester course",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: sharedErrors.ErrInternal.Message,
			Code:  sharedErrors.ErrInternal.Code,
		})
		return
	}

	reqLogger.Info("semester course deleted successfully",
		zap.String("course_code", response.CourseCode),
	)

	c.JSON(http.StatusOK, response)
}
