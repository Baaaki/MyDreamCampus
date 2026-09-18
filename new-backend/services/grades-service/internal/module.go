// Package grades wires the grades service's dependencies and exposes the
// platform-level Module + lifecycle hooks cmd/main.go uses.
package grades

import (
	"context"

	"github.com/baaaki/mydreamcampus/grades/internal/handler"
	"github.com/baaaki/mydreamcampus/grades/internal/repository"
	"github.com/baaaki/mydreamcampus/grades/internal/service"
	"github.com/baaaki/mydreamcampus/grades/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/rabbitmq"
	platformRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
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
	retentionStore   *repository.RetentionStore

	gradeService *service.GradeService

	gradeHandler *handler.GradeHandler

	eventConsumer    *worker.EventConsumer
	finalizeConsumer *worker.FinalizeConsumer
	periodConsumer   *worker.PeriodConsumer
}

func New(
	pool *pgxpool.Pool,
	rabbitConn *rabbitmq.Connection,
	periodRepo *platformRepo.SimplePeriodRepository,
	semesterClient service.SemesterClient,
) *Module {
	cacheRepo := repository.NewCacheRepository(pool)
	registrationRepo := repository.NewRegistrationRepository(pool)
	scoreRepo := repository.NewScoreRepository(pool)
	completedRepo := repository.NewCompletedRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)

	// Audit entries leave through this module's outbox; catalog consumes
	// them and owns the row. No cross-module write any more.
	auditLogger := audit.NewEventAuditLogger(repository.NewAuditOutbox(outboxRepo), "grades")

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
		retentionStore:   repository.NewRetentionStore(pool),
		gradeService:     gradeSvc,
		gradeHandler:     handler.NewGradeHandler(gradeSvc, studentGradeSvc),
		eventConsumer:    worker.NewEventConsumer(rabbitmq.NewConsumer(rabbitConn), cacheRepo, registrationRepo),
		finalizeConsumer: worker.NewFinalizeConsumer(rabbitmq.NewConsumer(rabbitConn), gradeSvc, completedRepo),
		periodConsumer:   worker.NewPeriodConsumer(rabbitmq.NewConsumer(rabbitConn), periodRepo),
	}
}

// Name is the URL slug under /api. Frontend already calls /api/grades.
func (m *Module) Name() string { return "grades" }

// OutboxStore for the outbox worker.
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// RetentionStore exposes this schema's event tables to the retention worker.
func (m *Module) RetentionStore() eventbus.RetentionStore { return m.retentionStore }

// Bootstrap starts the RabbitMQ consumers: sync events (student/course/
// enrollment/attendance projections), the finalize self-loop and the
// academic-period projection. Queue bindings are declared in cmd/main.go so
// events published before this point are not lost.
func (m *Module) Bootstrap(ctx context.Context) error {
	if err := m.eventConsumer.Start(ctx); err != nil {
		return err
	}
	if err := m.finalizeConsumer.Start(ctx); err != nil {
		return err
	}
	return m.periodConsumer.Start(ctx)
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
			// A retried score write would overwrite or duplicate marks that
			// the student has already been shown.
			teacher.POST("/courses/:course_id/scores", platformMiddleware.Idempotency(), m.gradeHandler.SubmitScore)
			teacher.POST("/courses/:course_id/scores/bulk", platformMiddleware.Idempotency(), m.gradeHandler.BulkSubmitScores)
			teacher.POST("/courses/:course_id/scores/lock", m.gradeHandler.LockAssessment)
		}

		// Student facing routes
		student := rg.Group("/my")
		student.Use(platformMiddleware.RequireStudent())
		{
			student.GET("/grades", m.gradeHandler.GetMyGrades)
			student.GET("/transcript", m.gradeHandler.GetTranscript)
		}

		// Admin facing routes. The appeal handler mutates a finalized
		// grade, so it must never sit behind RequireStudent — the handler
		// double-checks the admin role itself.
		admin := rg.Group("/admin")
		admin.Use(platformMiddleware.RequireAdmin())
		{
			admin.POST("/appeals", m.gradeHandler.ProcessAppeal)
			// Single-score lock/unlock takes a bare registration_id and skips
			// the course-instructor check, and a lock can trigger finalize —
			// under the teacher group any teacher could close any course.
			admin.POST("/scores/lock", m.gradeHandler.LockScore)
			admin.POST("/scores/unlock", m.gradeHandler.UnlockScore)
		}
	}
}
