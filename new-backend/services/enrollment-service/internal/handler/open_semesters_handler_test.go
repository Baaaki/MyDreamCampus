package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/enrollment/internal/db"
	"github.com/baaaki/mydreamcampus/enrollment/internal/service"
	"github.com/baaaki/mydreamcampus/shared/platform/clock/clocktest"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPeriodStore struct {
	periods []sharedRepo.SimplePeriod
	err     error
}

func (s *stubPeriodStore) GetActivePeriodBySemester(ctx context.Context, semester string) (*sharedRepo.SimplePeriod, error) {
	return nil, errors.New("not used")
}

func (s *stubPeriodStore) GetAllPeriods(ctx context.Context) ([]sharedRepo.SimplePeriod, error) {
	return s.periods, s.err
}

type stubProgramStore struct {
	service.EnrollmentStore
	programs []db.EnrollmentProgram
}

func (s *stubProgramStore) GetEnrollmentProgramsByStudent(ctx context.Context, studentID uuid.UUID, semester *string, status *string) ([]db.EnrollmentProgram, error) {
	return s.programs, nil
}

func serveOpenSemesters(t *testing.T, periods service.PeriodStore, userID string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	_ = logger.Init("test")

	h := NewEnrollmentHandler(service.NewEnrollmentService(&stubProgramStore{}, nil, nil, nil, periods, nil))
	router := gin.New()
	router.GET("/open-semesters", func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
	}, h.GetOpenSemesters)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/open-semesters", nil))
	return resp
}

func TestGetOpenSemesters_WindowOpen_ReturnsArray(t *testing.T) {
	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	clocktest.Freeze(t, now)
	periods := &stubPeriodStore{periods: []sharedRepo.SimplePeriod{{
		Semester:    "2025-2026-Spring",
		PeriodStart: now.AddDate(0, 0, -1),
		PeriodEnd:   now.AddDate(0, 0, 6),
		IsActive:    true,
	}}}

	resp := serveOpenSemesters(t, periods, uuid.NewString())

	require.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `[{
		"semester": "2025-2026-Spring",
		"enrollment_open": true,
		"period_start": "2026-02-09T12:00:00Z",
		"period_end": "2026-02-16T12:00:00Z",
		"has_program": false
	}]`, resp.Body.String())
}

func TestGetOpenSemesters_NothingOpen_ReturnsEmptyArray(t *testing.T) {
	resp := serveOpenSemesters(t, &stubPeriodStore{}, uuid.NewString())

	require.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `[]`, resp.Body.String())
}

func TestGetOpenSemesters_NoUserInContext_Returns401(t *testing.T) {
	resp := serveOpenSemesters(t, &stubPeriodStore{}, "")

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestGetOpenSemesters_PeriodStoreFails_Returns500(t *testing.T) {
	resp := serveOpenSemesters(t, &stubPeriodStore{err: errors.New("db down")}, uuid.NewString())

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}
