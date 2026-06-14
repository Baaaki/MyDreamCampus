// Package grades wires the grades module's dependencies and
// exposes the platform-level Module + lifecycle hooks main.go uses.
package grades

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/handler"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/service"
	platformMiddleware "github.com/baaaki/mydreamcampus/monolith/internal/platform/middleware"
	platformRepo "github.com/baaaki/mydreamcampus/monolith/internal/platform/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/audit"
	ccService "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	pool *pgxpool.Pool

	cacheRepo        *repository.CacheRepository
	registrationRepo *repository.RegistrationRepository
	scoreRepo        *repository.ScoreRepository
	completedRepo    *repository.CompletedRepository
	outboxRepo       *repository.OutboxRepository
	periodRepo       *platformRepo.SimplePeriodRepository
	outboxStore      *repository.OutboxStore

	gradeService *service.GradeService

	gradeHandler *handler.GradeHandler
}

func New(
	pool *pgxpool.Pool,
	periodRepo *platformRepo.SimplePeriodRepository,
	auditLogger audit.Logger,
	semesterSvc *ccService.SemesterService,
) *Module {
	cacheRepo := repository.NewCacheRepository(pool)
	registrationRepo := repository.NewRegistrationRepository(pool)
	scoreRepo := repository.NewScoreRepository(pool)
	completedRepo := repository.NewCompletedRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)

	semesterClient := service.NewInProcessSemesterClient(semesterSvc)

	gradeSvc := service.NewGradeService(
		pool,
		cacheRepo,
		registrationRepo,
		scoreRepo,
		completedRepo,
		outboxRepo,
		periodRepo,
		auditLogger,
		semesterClient,
	)

	studentGradeSvc := service.NewStudentGradesService(
		cacheRepo,
		registrationRepo,
		scoreRepo,
		completedRepo,
	)

	return &Module{
		pool:             pool,
		cacheRepo:        cacheRepo,
		registrationRepo: registrationRepo,
		scoreRepo:        scoreRepo,
		completedRepo:    completedRepo,
		outboxRepo:       outboxRepo,
		periodRepo:       periodRepo,
		outboxStore:      repository.NewOutboxStore(outboxRepo),
		gradeService:     gradeSvc,
		gradeHandler:     handler.NewGradeHandler(gradeSvc, studentGradeSvc),
	}
}

// Name is the URL slug under /api. Frontend already calls /api/grades.
func (m *Module) Name() string { return "grades" }

// OutboxStore for the per-module outbox worker.
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// Bootstrap starts the staff/student/course event consumers
func (m *Module) Bootstrap(ctx context.Context) error {
	// Note: Currently consumers are not wired here yet, they run as separate generic workers
	return nil
}

// RegisterRoutes mounts /api/grades/*. All routes JWT-authed.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Use(platformMiddleware.JWTAuth())
	rg.Use(platformMiddleware.CSRFProtection())
	rg.Use(platformMiddleware.UserRateLimit())
	{
		// Teacher facing routes
		teacher := rg.Group("")
		teacher.Use(platformMiddleware.RequireRole("teacher", "admin"))
		{
			teacher.GET("/courses/:course_id/status", m.gradeHandler.GetCourseStatus)
			teacher.GET("/courses/:course_id/students", m.gradeHandler.GetCourseStudents)
			teacher.POST("/courses/:course_id/scores", m.gradeHandler.SubmitScore)
			teacher.POST("/courses/:course_id/scores/bulk", m.gradeHandler.BulkSubmitScores)
			teacher.POST("/courses/:course_id/scores/lock", m.gradeHandler.LockAssessment)
			teacher.POST("/courses/:course_id/scores/:slug/lock", m.gradeHandler.LockScore)
			teacher.POST("/courses/:course_id/scores/:slug/unlock", m.gradeHandler.UnlockScore)
		}

		// Student facing routes
		student := rg.Group("/my")
		student.Use(platformMiddleware.RequireStudent())
		{
			student.GET("/grades", m.gradeHandler.GetMyGrades)
			student.GET("/transcript", m.gradeHandler.GetTranscript)
			student.POST("/appeals", m.gradeHandler.ProcessAppeal)
		}
	}
}
