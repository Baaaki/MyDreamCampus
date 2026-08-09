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

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/rabbitmq"
	"github.com/baaaki/mydreamcampus/student/internal/handler"
	"github.com/baaaki/mydreamcampus/student/internal/repository"
	"github.com/baaaki/mydreamcampus/student/internal/service"
	"github.com/baaaki/mydreamcampus/student/internal/worker"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	cfg  *config.Config
	pool *pgxpool.Pool

	studentRepo         *repository.StudentRepository
	outboxRepo          *repository.OutboxRepository
	processedEventsRepo *repository.ProcessedEventsRepository
	importRepo          *repository.ImportRepository
	importJobsRepo      *repository.ImportJobsRepository
	outboxStore         *repository.OutboxStore
	retentionStore      *repository.RetentionStore

	studentService *service.StudentService
	importService  *service.ImportService
	staffClient    service.StaffServiceInterface

	studentHandler *handler.StudentHandler
	consumer       *worker.EventConsumer
}

// New wires the module from shared infra. The staff client comes from
// main.go so the advisor lookup can run either in-process or over internal
// REST without the module knowing which.
func New(
	cfg *config.Config,
	pool *pgxpool.Pool,
	rabbitConn *rabbitmq.Connection,
	staffClient service.StaffServiceInterface,
) *Module {
	studentRepo := repository.NewStudentRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	processedEventsRepo := repository.NewProcessedEventsRepository(pool)
	importRepo := repository.NewImportRepository(pool)
	importJobsRepo := repository.NewImportJobsRepository(pool)

	studentSvc := service.NewStudentService(studentRepo, staffClient)
	importSvc := service.NewImportService(importRepo, studentRepo, staffClient)

	consumer := worker.NewEventConsumer(rabbitmq.NewConsumer(rabbitConn), studentRepo, processedEventsRepo)

	return &Module{
		cfg:                 cfg,
		pool:                pool,
		studentRepo:         studentRepo,
		outboxRepo:          outboxRepo,
		processedEventsRepo: processedEventsRepo,
		importRepo:          importRepo,
		importJobsRepo:      importJobsRepo,
		outboxStore:         repository.NewOutboxStore(outboxRepo),
		retentionStore:      repository.NewRetentionStore(pool),
		studentService:      studentSvc,
		importService:       importSvc,
		staffClient:         staffClient,
		studentHandler:      handler.NewStudentHandler(studentSvc, importSvc),
		consumer:            consumer,
	}
}

// Name is the URL slug under /api. Stays plural ("students") to match the
// legacy microservice URL the frontend already calls — module identity
// (schema, package) remains singular.
func (m *Module) Name() string { return "students" }

// StudentService exposes the internal service for cross-module calls.
func (m *Module) StudentService() *service.StudentService { return m.studentService }

// OutboxStore for the per-module outbox worker.
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// RetentionStore exposes this schema's event tables to the retention worker.
func (m *Module) RetentionStore() eventbus.RetentionStore { return m.retentionStore }

// Bootstrap starts the staff-events consumer. Once staff is in the same
// process this can move to in-process pubsub — keeping
// RabbitMQ for now mirrors the legacy contract and lets us migrate
// modules incrementally.
func (m *Module) Bootstrap(ctx context.Context) error {
	return m.consumer.Start(ctx)
}

// RegisterRoutes mounts /api/students/*. All routes are JWT-protected;
// admin-only ones get an extra RequireAdmin(). Enrollment's reads live on
// the root /internal tree — see RegisterPublicRoutes.
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
			// A retried create must not produce a second student record.
			admin.POST("", platformMiddleware.Idempotency(), m.studentHandler.CreateStudent)
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

// RegisterPublicRoutes mounts the internal reads enrollment performs. They
// sit at the root rather than under /api because Caddy only proxies /api,
// which puts them out of reach from outside the compose network. The secret
// check stays behind that as the second line of defence.
func (m *Module) RegisterPublicRoutes(r *gin.Engine) {
	internal := r.Group("/internal")
	internal.Use(platformMiddleware.RequireInternalSecret(m.cfg.Server.InternalSecret))
	{
		internal.GET("/students/:id", m.studentHandler.GetStudentByID)
		internal.GET("/students", m.studentHandler.ListStudentsByAdvisor)
	}
}
