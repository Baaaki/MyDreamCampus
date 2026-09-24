// Package course_catalog wires the course catalog module's dependencies
// and exposes the platform-level Module + lifecycle hooks main.go uses.
//
// Owns the course_catalog schema (course_catalog table, semester_courses,
// course_schedule_sessions, semesters, audit_log, academic_periods,
// outbox_events) and publishes course.* events through its outbox.
package coursecatalog

import (
	"context"

	"github.com/baaaki/mydreamcampus/catalog/internal/handler"
	"github.com/baaaki/mydreamcampus/catalog/internal/repository"
	"github.com/baaaki/mydreamcampus/catalog/internal/service"
	"github.com/baaaki/mydreamcampus/catalog/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/clocksync"
	platformHandler "github.com/baaaki/mydreamcampus/shared/platform/handler"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/rabbitmq"
	platformRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	cfg  *config.Config
	pool *pgxpool.Pool

	catalogRepo        *repository.CatalogRepository
	semesterRepo       *repository.SemesterRepository
	scheduleRepo       *repository.ScheduleRepository
	outboxRepo         *repository.OutboxRepository
	auditRepo          *repository.AuditRepository
	semesterStatusRepo *repository.SemesterStatusRepository
	periodRepo         *platformRepo.SimplePeriodRepository
	outboxStore        *repository.OutboxStore
	retentionStore     *repository.RetentionStore

	auditLogger     audit.Logger
	staffClient     service.StaffClient
	catalogService  *service.CatalogService
	semesterService *service.SemesterService

	catalogHandler        *handler.CatalogHandler
	semesterHandler       *handler.SemesterHandler
	semesterStatusHandler *handler.SemesterStatusHandler
	auditHandler          *handler.AuditHandler
	periodHandler         *platformHandler.SimplePeriodHandler
	timeHandler           *platformHandler.TimeControlHandler

	auditConsumer *worker.AuditConsumer
}

// New constructs the catalog service. The staff and meal clients come from
// main.go, so the wiring root owns the transports and this package only
// sees interfaces. clockBackend is nil when Redis is unavailable.
func New(
	cfg *config.Config,
	pool *pgxpool.Pool,
	rabbitConn *rabbitmq.Connection,
	staffClient service.StaffClient,
	mealClient service.MealClient,
	clockBackend clocksync.Backend,
) *Module {
	catalogRepo := repository.NewCatalogRepository(pool)
	semesterRepo := repository.NewSemesterRepository(pool)
	scheduleRepo := repository.NewScheduleRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	periodRepo := platformRepo.NewSimplePeriodRepository(pool, "course_catalog")

	auditLogger := service.NewDirectAuditLogger(auditRepo, "course-catalog")
	semesterStatusRepo := repository.NewSemesterStatusRepository(pool, auditLogger)

	catalogSvc := service.NewCatalogService(catalogRepo)
	semesterSvc := service.NewSemesterService(
		catalogRepo, semesterRepo, scheduleRepo, outboxRepo,
		staffClient, periodRepo, semesterStatusRepo,
	)

	return &Module{
		cfg:                cfg,
		pool:               pool,
		catalogRepo:        catalogRepo,
		semesterRepo:       semesterRepo,
		scheduleRepo:       scheduleRepo,
		outboxRepo:         outboxRepo,
		auditRepo:          auditRepo,
		semesterStatusRepo: semesterStatusRepo,
		periodRepo:         periodRepo,
		outboxStore:        repository.NewOutboxStore(outboxRepo),
		retentionStore:     repository.NewRetentionStore(pool),
		auditLogger:        auditLogger,
		staffClient:        staffClient,
		catalogService:     catalogSvc,
		semesterService:    semesterSvc,
		catalogHandler:     handler.NewCatalogHandler(catalogSvc),
		semesterHandler:    handler.NewSemesterHandler(semesterSvc),
		semesterStatusHandler: handler.NewSemesterStatusHandler(
			semesterStatusRepo, periodRepo, auditLogger, mealClient, pool,
		),
		auditHandler:  handler.NewAuditHandler(auditRepo),
		auditConsumer: worker.NewAuditConsumer(rabbitmq.NewConsumer(rabbitConn), auditRepo),
		periodHandler: platformHandler.NewSimplePeriodHandler(periodRepo, semesterStatusRepo, auditLogger),
		timeHandler:   platformHandler.NewTimeControlHandler("catalog", clockBackend, auditLogger),
	}
}

// Bootstrap starts the audit consumer: grades and meal publish their audit
// entries now instead of writing into this module's table.
func (m *Module) Bootstrap(ctx context.Context) error {
	return m.auditConsumer.Start(ctx)
}

// Name is the URL slug under /api. The schema is `course_catalog`, but the
// clients call `/api/catalog`, so the slug keeps the shorter name.
func (m *Module) Name() string { return "catalog" }

// OutboxStore for the outbox worker.
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// RetentionStore exposes this schema's event tables to the retention worker.
func (m *Module) RetentionStore() eventbus.RetentionStore { return m.retentionStore }

// RegisterRoutes mounts /api/catalog/*. Public endpoints (anonymous
// course browsing) live before the JWT-auth chain so the public router
// matches first.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	// Public — anonymous browsing of catalog courses.
	rg.GET("/courses", m.catalogHandler.ListCourses)
	rg.GET("/courses/:course_code", m.catalogHandler.GetCourseByCourseCode)

	// Protected — JWT + CSRF + per-user rate limit.
	protected := rg.Group("")
	protected.Use(platformMiddleware.JWTAuth())
	protected.Use(platformMiddleware.CSRFProtection())
	protected.Use(platformMiddleware.UserRateLimit())
	{
		protected.POST("/courses", platformMiddleware.RequireAdmin(), m.catalogHandler.CreateCourse)
		protected.PUT("/courses/:course_code", platformMiddleware.RequireAdmin(), m.catalogHandler.UpdateCourse)

		// Admin-only group — time machine controls (every service serves its
		// own status), periods, semester status, audit log.
		admin := protected.Group("/admin")
		admin.Use(platformMiddleware.RequireAdmin())
		{
			m.timeHandler.RegisterRoutes(admin)
			m.periodHandler.RegisterRoutes(admin)
			m.semesterStatusHandler.RegisterRoutes(admin)
			m.auditHandler.RegisterAdminRoutes(admin)
		}
	}

}

// RegisterPublicRoutes mounts the legacy /api/semesters routes that the
// frontend uses for course-code-by-semester lookups. They live outside
// /api/catalog because the frontend predates the modular layout.
func (m *Module) RegisterPublicRoutes(r *gin.Engine) {
	// Service-to-service semester lookups, the audit sink grades and meal
	// write to, and the period republish hook. At the root rather than under
	// /api: Caddy only proxies /api, so these cannot be reached from outside
	// the compose network. RequireInternalSecret stays as the second line.
	internal := r.Group("/internal")
	internal.Use(platformMiddleware.RequireInternalSecret(m.cfg.Server.InternalSecret))
	{
		m.semesterStatusHandler.RegisterInternalRoutes(internal)
		m.semesterHandler.RegisterInternalRoutes(internal)
		m.auditHandler.RegisterInternalRoutes(internal)
	}

	semesters := r.Group("/api/semesters")
	semesters.Use(platformMiddleware.JWTAuth())
	semesters.Use(platformMiddleware.CSRFProtection())
	semesters.Use(platformMiddleware.UserRateLimit())
	{
		semesters.GET("/teacher/courses",
			platformMiddleware.RequireRole("teacher"),
			m.semesterHandler.GetTeacherCourses,
		)

		semesterCourses := semesters.Group("/:semester_id/courses")
		{
			semesterCourses.GET("", m.semesterHandler.ListSemesterCourses)
			semesterCourses.GET("/:course_id", m.semesterHandler.GetSemesterCourseByID)
			semesterCourses.POST("", platformMiddleware.RequireAdmin(), m.semesterHandler.CreateSemesterCourse)
			semesterCourses.DELETE("/:course_id", platformMiddleware.RequireAdmin(), m.semesterHandler.DeleteSemesterCourse)
		}
	}
}
