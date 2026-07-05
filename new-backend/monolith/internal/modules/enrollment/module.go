// Package enrollment wires the enrollment module's dependencies and
// exposes the platform-level Module + lifecycle hooks main.go uses.
//
// Owns the enrollment schema (students_cache, semester_courses_cache,
// course_sessions_cache, student_passed_prerequisites, enrollment_programs,
// enrollment_program_courses, enrollment_rejection_logs, outbox_events,
// processed_events). Reads academic_periods from the course_catalog
// schema via platform/repository.SimplePeriodRepository.
package enrollment

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/handler"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/worker"
	platformMiddleware "github.com/baaaki/mydreamcampus/monolith/internal/platform/middleware"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	platformRepo "github.com/baaaki/mydreamcampus/monolith/internal/platform/repository"
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

	enrollmentService *service.EnrollmentService

	enrollmentHandler *handler.EnrollmentHandler

	eventConsumer *worker.EventConsumer
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
		enrollmentService:   enrollmentSvc,
		enrollmentHandler:   handler.NewEnrollmentHandler(enrollmentSvc),
		eventConsumer:       worker.NewEventConsumer(rabbitmq.NewConsumer(rabbitConn), passedPrereqRepo),
	}
}

// Name is the URL slug under /api. Frontend already calls /api/enrollment.
func (m *Module) Name() string { return "enrollment" }

// OutboxStore for the per-module outbox worker.
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// Bootstrap starts the RabbitMQ consumer feeding the passed-prerequisite
// projection. Queue bindings are pre-declared in main.go so events published
// before this point are not lost.
func (m *Module) Bootstrap(ctx context.Context) error {
	return m.eventConsumer.Start(ctx)
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
			student.POST("/programs", m.enrollmentHandler.CreateEnrollmentProgram)
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
			advisor.POST("/programs/:program_id/approve", m.enrollmentHandler.ApproveEnrollmentProgram)
			advisor.POST("/programs/:program_id/reject", m.enrollmentHandler.RejectEnrollmentProgram)
		}
	}
}
