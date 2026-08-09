// Package enrollment wires the enrollment service's dependencies and
// exposes the platform-level Module + lifecycle hooks cmd/main.go uses.
//
// Owns the enrollment database (students_cache, semester_courses_cache,
// course_sessions_cache, student_passed_prerequisites, enrollment_programs,
// enrollment_program_courses, enrollment_rejection_logs, outbox_events,
// processed_events, academic_periods). academic_periods is a projection of
// catalog's enrollment-typed period, fed by PeriodConsumer.
package enrollment

import (
	"context"

	"github.com/baaaki/mydreamcampus/enrollment/internal/handler"
	"github.com/baaaki/mydreamcampus/enrollment/internal/repository"
	"github.com/baaaki/mydreamcampus/enrollment/internal/service"
	"github.com/baaaki/mydreamcampus/enrollment/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/rabbitmq"
	platformRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	pool *pgxpool.Pool

	enrollmentRepo      *repository.EnrollmentRepository
	outboxRepo          *repository.OutboxRepository
	processedEventsRepo *repository.ProcessedEventsRepository
	passedPrereqRepo    *repository.PassedPrerequisitesRepository
	periodRepo          *platformRepo.SimplePeriodRepository
	outboxStore         *repository.OutboxStore
	retentionStore      *repository.RetentionStore

	enrollmentService *service.EnrollmentService

	enrollmentHandler *handler.EnrollmentHandler

	eventConsumer  *worker.EventConsumer
	periodConsumer *worker.PeriodConsumer
}

func New(
	pool *pgxpool.Pool,
	rabbitConn *rabbitmq.Connection,
	studentClient service.StudentClient,
	courseCatalogClient service.CourseCatalogClient,
	periodRepo *platformRepo.SimplePeriodRepository,
) *Module {
	enrollmentRepo := repository.NewEnrollmentRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	processedEventsRepo := repository.NewProcessedEventsRepository(pool)
	passedPrereqRepo := repository.NewPassedPrerequisitesRepository(pool)

	enrollmentSvc := service.NewEnrollmentService(enrollmentRepo, passedPrereqRepo, studentClient, courseCatalogClient, periodRepo)

	return &Module{
		pool:                pool,
		enrollmentRepo:      enrollmentRepo,
		outboxRepo:          outboxRepo,
		processedEventsRepo: processedEventsRepo,
		passedPrereqRepo:    passedPrereqRepo,
		periodRepo:          periodRepo,
		outboxStore:         repository.NewOutboxStore(outboxRepo),
		retentionStore:      repository.NewRetentionStore(pool),
		enrollmentService:   enrollmentSvc,
		enrollmentHandler:   handler.NewEnrollmentHandler(enrollmentSvc),
		eventConsumer:       worker.NewEventConsumer(rabbitmq.NewConsumer(rabbitConn), passedPrereqRepo),
		periodConsumer:      worker.NewPeriodConsumer(rabbitmq.NewConsumer(rabbitConn), periodRepo),
	}
}

// Name is the URL slug under /api. Frontend already calls /api/enrollment.
func (m *Module) Name() string { return "enrollment" }

// OutboxStore for the outbox worker.
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// RetentionStore exposes this schema's event tables to the retention worker.
func (m *Module) RetentionStore() eventbus.RetentionStore { return m.retentionStore }

// Bootstrap starts the RabbitMQ consumers feeding the passed-prerequisite and
// academic-period projections. Queue bindings are declared in cmd/main.go so
// events published before this point are not lost.
func (m *Module) Bootstrap(ctx context.Context) error {
	if err := m.eventConsumer.Start(ctx); err != nil {
		return err
	}
	return m.periodConsumer.Start(ctx)
}

// RegisterRoutes mounts /api/enrollment/*. All routes JWT-authed.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Use(platformMiddleware.JWTAuth())
	rg.Use(platformMiddleware.CSRFProtection())
	rg.Use(platformMiddleware.UserRateLimit())
	{
		// Student-facing — viewing and managing one's own enrollment.
		student := rg.Group("")
		student.Use(platformMiddleware.RequireStudent())
		{
			student.GET("/available-courses", m.enrollmentHandler.GetAvailableCourses)
			// Submitting a program consumes course quota, so a retry must
			// not book a second seat.
			student.POST("/programs", platformMiddleware.Idempotency(), m.enrollmentHandler.CreateEnrollmentProgram)
			student.DELETE("/programs", m.enrollmentHandler.CancelMyEnrollment)
			student.GET("/my-enrollments", m.enrollmentHandler.GetMyEnrollments)
			student.GET("/latest-rejection", m.enrollmentHandler.GetLatestRejection)
			student.GET("/my-rejections", m.enrollmentHandler.GetMyRejections)
		}

		// Advisor — review submitted programs.
		advisor := rg.Group("/advisor")
		advisor.Use(platformMiddleware.RequireRole("teacher", "admin"))
		{
			advisor.GET("/pending-programs", m.enrollmentHandler.GetPendingProgramsByAdvisor)
			// A double approve or reject would emit the decision event twice.
			advisor.POST("/programs/:program_id/approve", platformMiddleware.Idempotency(), m.enrollmentHandler.ApproveEnrollmentProgram)
			advisor.POST("/programs/:program_id/reject", platformMiddleware.Idempotency(), m.enrollmentHandler.RejectEnrollmentProgram)
		}
	}
}
