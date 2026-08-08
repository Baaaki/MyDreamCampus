// Package course_catalog wires the course catalog module's dependencies
// and exposes the platform-level Module + lifecycle hooks main.go uses.
//
// Owns the course_catalog schema (course_catalog table, semester_courses,
// course_schedule_sessions, semesters, audit_log, academic_periods,
// outbox_events) and publishes course.* events through its outbox.
package coursecatalog

import (
	"github.com/baaaki/mydreamcampus/monolith/config"
	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/handler"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	platformHandler "github.com/baaaki/mydreamcampus/shared/platform/handler"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
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

	auditLogger     audit.Logger
	staffClient     service.StaffClient
	catalogService  *service.CatalogService
	semesterService *service.SemesterService

	catalogHandler        *handler.CatalogHandler
	semesterHandler       *handler.SemesterHandler
	semesterStatusHandler *handler.SemesterStatusHandler
	auditHandler          *handler.AuditHandler
	periodHandler         *platformHandler.SimplePeriodHandler
	timeHandler           *platformHandler.TimeHandler
}

// New constructs the course catalog module. The staff and meal clients come
// from main.go so the same module runs against in-process adapters or the
// internal REST ones.
func New(
	cfg *config.Config,
	pool *pgxpool.Pool,
	staffClient service.StaffClient,
	mealClient service.MealClient,
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
		periodHandler: platformHandler.NewSimplePeriodHandler(periodRepo, semesterStatusRepo, auditLogger),
		timeHandler:   platformHandler.NewTimeHandler(),
	}
}

// Name is the URL slug under /api. Plan section 0.2 names the module
// `course_catalog`; the legacy URL prefix is `/api/catalog` and the
// frontend depends on it, so we keep that here.
func (m *Module) Name() string { return "catalog" }

// OutboxStore for the per-module outbox worker.
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// SemesterService is the in-process handle the enrollment / grades /
// attendance modules use.
func (m *Module) SemesterService() *service.SemesterService { return m.semesterService }

// CatalogService is the in-process handle other modules use for course
// metadata lookups.
func (m *Module) CatalogService() *service.CatalogService { return m.catalogService }

// AuditRepo provides the shared audit repository to other modules.
func (m *Module) AuditRepo() *repository.AuditRepository { return m.auditRepo }

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

		// Admin-only group — Time Machine, periods, semester status, audit log.
		admin := protected.Group("/admin")
		admin.Use(platformMiddleware.RequireAdmin())
		{
			m.timeHandler.RegisterRoutes(admin)
			m.periodHandler.RegisterRoutes(admin)
			m.semesterStatusHandler.RegisterRoutes(admin)
			m.auditHandler.RegisterAdminRoutes(admin)
		}
	}

	// Internal sub-tree — service-to-service semester lookups plus the
	// period republish hook used to backfill cold consumer projections.
	// Shared-secret auth keeps them off the public surface.
	internal := rg.Group("/internal")
	internal.Use(platformMiddleware.RequireInternalSecret(m.cfg.Server.InternalSecret))
	{
		m.semesterStatusHandler.RegisterInternalRoutes(internal)
		m.semesterHandler.RegisterInternalRoutes(internal)
		m.auditHandler.RegisterInternalRoutes(internal)
	}
}

// RegisterPublicRoutes mounts the legacy /api/semesters routes that the
// frontend uses for course-code-by-semester lookups. They live outside
// /api/catalog because the frontend predates the modular layout.
func (m *Module) RegisterPublicRoutes(r *gin.Engine) {
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
