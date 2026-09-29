//go:build integration

package grades

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/grades/internal/db"
	gradeErrors "github.com/baaaki/mydreamcampus/grades/internal/errors"
	"github.com/baaaki/mydreamcampus/grades/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// advisorsByStudent stands in for the student service's advisor records.
type advisorsByStudent map[uuid.UUID]uuid.UUID

func (a advisorsByStudent) GetStudentByID(_ context.Context, id uuid.UUID) (contracts.StudentResponse, error) {
	advisor, ok := a[id]
	if !ok {
		return contracts.StudentResponse{ID: id.String()}, nil
	}
	advisorID := advisor.String()
	return contracts.StudentResponse{ID: id.String(), AdvisorID: &advisorID}, nil
}

func bearer(t *testing.T, userID uuid.UUID, role string) string {
	t.Helper()
	tok, _, err := utils.GenerateAccessToken(userID.String(), role, "", 0)
	require.NoError(t, err)
	return "Bearer " + tok
}

// The module's real routes and middleware over a real database: one
// advisee with a finalized course, and a student nobody advises.
func TestTranscriptRoutes_WhoGetsWhichTranscript(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	utils.InitJWTSecret("test-secret-key-minimum-32-characters-long-aaa")
	platformMiddleware.SetBlacklistChecker(nil)
	ctx := context.Background()
	pool := pgtest.New(t, "grades", "sql/migrations")

	advisor, advisee, unadvised := uuid.New(), uuid.New(), uuid.New()
	cache := repository.NewCacheRepository(pool)
	for i, id := range []uuid.UUID{advisee, unadvised} {
		_, err := cache.UpsertStudentCache(ctx, db.UpsertStudentCacheParams{
			ID: id, StudentNumber: []string{"2026001", "2026002"}[i], IsActive: pgtype.Bool{Bool: true, Valid: true},
		})
		require.NoError(t, err)
	}
	_, err := repository.NewCompletedRepository(pool).CreateCompletedCourse(ctx, db.CreateCompletedCourseParams{
		StudentID: advisee, StudentNumber: "2026001", CourseID: uuid.New(), CourseCode: "BLM101", CourseName: "Programlamaya Giriş",
		Credits: 4, Semester: "2025-2026-Spring", InstructorID: advisor, AssessmentScores: []byte(`{}`),
		WeightedAverage: utils.Float64ToPgNumeric(85), GradePoint: db.GradesGradePointEnum400, GradingType: db.GradesGradingTypeEnumAbsolute,
		GradingConfig: []byte(`{}`), ClassStatistics: []byte(`{}`), FinalizedAt: utils.TimeToPgTimestamp(time.Now()), FinalizedBy: advisor,
	})
	require.NoError(t, err)

	m := New(pool, nil, sharedRepo.NewSimplePeriodRepository(pool, "grades"), nil, advisorsByStudent{advisee: advisor})
	router := gin.New()
	m.RegisterRoutes(router.Group("/api/grades"))

	of := func(id uuid.UUID) string { return "/api/grades/students/" + id.String() + "/transcript" }
	cases := []struct {
		name, path, auth string
		want             int
		code             string
	}{
		{"student reads their own", "/api/grades/my/transcript", bearer(t, advisee, "student"), http.StatusOK, ""},
		{"advisor reads an advisee", of(advisee), bearer(t, advisor, "teacher"), http.StatusOK, ""},
		{"admin reads anyone", of(unadvised), bearer(t, uuid.New(), "admin"), http.StatusOK, ""},
		{"another teacher is refused", of(advisee), bearer(t, uuid.New(), "teacher"), http.StatusForbidden, gradeErrors.ErrNotStudentAdvisor.Code},
		{"advisor of someone else is refused", of(unadvised), bearer(t, advisor, "teacher"), http.StatusForbidden, gradeErrors.ErrNotStudentAdvisor.Code},
		{"a student cannot use the staff route, not even for their own id", of(advisee), bearer(t, advisee, "student"), http.StatusForbidden, ""},
		{"a teacher has no transcript of their own", "/api/grades/my/transcript", bearer(t, advisor, "teacher"), http.StatusForbidden, ""},
		{"no token", of(advisee), "", http.StatusUnauthorized, ""},
		{"malformed student id", "/api/grades/students/nope/transcript", bearer(t, advisor, "teacher"), http.StatusBadRequest, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, c.path, nil)
			if c.auth != "" {
				req.Header.Set("Authorization", c.auth)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, c.want, w.Code, w.Body.String())
			if c.code != "" {
				assert.Contains(t, w.Body.String(), c.code)
			}
			if c.want == http.StatusOK && c.path != of(unadvised) {
				assert.Contains(t, w.Body.String(), "BLM101")
			}
		})
	}
}
