// Package student wires the student module's dependencies and exposes
// the platform-level Module + lifecycle hooks main.go consumes.
//
// The module owns the student schema (students, outbox_events,
// processed_events, import_jobs). It publishes student.created/updated/
// deactivated events through its outbox and consumes staff.deactivated
// to drop advisor assignments.
package student

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/student/handler"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/student/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/student/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/student/worker"
	platformMiddleware "github.com/baaaki/mydreamcampus/monolith/internal/platform/middleware"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	staffService "github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	pool *pgxpool.Pool

	studentRepo         *repository.StudentRepository
	outboxRepo          *repository.OutboxRepository
	processedEventsRepo *repository.ProcessedEventsRepository
	importRepo          *repository.ImportRepository
	importJobsRepo      *repository.ImportJobsRepository
	outboxStore         *repository.OutboxStore

	studentService *service.StudentService
	importService  *service.ImportService
	staffClient    *service.StaffClient

	studentHandler *handler.StudentHandler
	consumer       *worker.EventConsumer
}

// New wires the module from shared infra. The staff service handle is
// passed in so cross-module reads (advisor lookup) happen in-process
// instead of via HTTP — see plan section 8 strategy 1.
func New(
	pool *pgxpool.Pool,
	rabbitConn *rabbitmq.Connection,
	staff *staffService.StaffService,
) *Module {
	studentRepo := repository.NewStudentRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	processedEventsRepo := repository.NewProcessedEventsRepository(pool)
	importRepo := repository.NewImportRepository(pool)
	importJobsRepo := repository.NewImportJobsRepository(pool)

	staffClient := service.NewStaffClient(staff)

	studentSvc := service.NewStudentService(studentRepo, staffClient)
	importSvc := service.NewImportService(importRepo, studentRepo, staffClient)

	consumer := worker.NewEventConsumer(rabbitmq.NewConsumer(rabbitConn), studentRepo, processedEventsRepo)

	return &Module{
		pool:                pool,
		studentRepo:         studentRepo,
		outboxRepo:          outboxRepo,
		processedEventsRepo: processedEventsRepo,
		importRepo:          importRepo,
		importJobsRepo:      importJobsRepo,
		outboxStore:         repository.NewOutboxStore(outboxRepo),
		studentService:      studentSvc,
		importService:       importSvc,
		staffClient:         staffClient,
		studentHandler:      handler.NewStudentHandler(studentSvc, importSvc),
		consumer:            consumer,
	}
}

// Name is the URL slug under /api. Stays plural ("students") to match the
// legacy microservice URL the frontend already calls — module identity
// (schema, package) remains singular per plan section 0.2.
func (m *Module) Name() string { return "students" }

// StudentService exposes the internal service for cross-module calls (Strateji 1).
func (m *Module) StudentService() *service.StudentService { return m.studentService }

// OutboxStore for the per-module outbox worker.
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// Bootstrap starts the staff-events consumer. Once staff is in the same
// process this can move to in-process pubsub (plan section 8) — keeping
// RabbitMQ for now mirrors the legacy contract and lets us migrate
// modules incrementally.
func (m *Module) Bootstrap(ctx context.Context) error {
	return m.consumer.Start(ctx)
}

// RegisterRoutes mounts /api/student/*. All routes are JWT-protected;
// admin-only ones get an extra RequireAdmin().
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Use(platformMiddleware.JWTAuth())
	rg.Use(platformMiddleware.CSRFProtection())
	rg.Use(platformMiddleware.UserRateLimit())
	{
		// Read endpoints — any authenticated user.
		rg.GET("", m.studentHandler.ListStudents)
		rg.POST("/search", m.studentHandler.SearchStudents)
		rg.GET("/:id", m.studentHandler.GetStudentByID)
		// Teacher/admin only — view their assigned advisees.
		rg.GET("/my-advisees",
			platformMiddleware.RequireRole("teacher", "admin"),
			m.studentHandler.GetMyAdvisees,
		)

		admin := rg.Group("")
		admin.Use(platformMiddleware.RequireAdmin())
		{
			admin.POST("", m.studentHandler.CreateStudent)
			admin.PUT("/:id", m.studentHandler.UpdateStudent)
			admin.DELETE("/:id", m.studentHandler.DeleteStudent)
			admin.GET("/orphaned", m.studentHandler.ListOrphanedStudents)
			admin.PUT("/bulk-advisor-assign", m.studentHandler.BulkAssignAdvisor)
			admin.POST("/bulk-import", m.studentHandler.BulkImport)
			admin.GET("/bulk-import/:job_id", m.studentHandler.GetImportJobStatus)
			admin.GET("/bulk-import", m.studentHandler.ListImportJobs)
		}
	}
}
