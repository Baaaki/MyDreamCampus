// Package staff wires the staff service's dependencies and exposes the
// platform-level Module + lifecycle hooks cmd/main.go consumes.
//
// The service owns the staff database (staff, outbox_events,
// teacher_profiles, admin_staff) and publishes staff.created/updated/
// deactivated events through its own outbox table. admin_staff is a
// directory, not accounts, and publishes nothing.
package staff

import (
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	platformHandler "github.com/baaaki/mydreamcampus/shared/platform/handler"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/staff/internal/handler"
	"github.com/baaaki/mydreamcampus/staff/internal/repository"
	"github.com/baaaki/mydreamcampus/staff/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Module is the staff module's wiring root.
type Module struct {
	cfg  *config.Config
	pool *pgxpool.Pool

	staffRepo          *repository.StaffRepository
	outboxRepo         *repository.OutboxRepository
	teacherProfileRepo *repository.TeacherProfileRepository
	adminStaffRepo     *repository.AdminStaffRepository
	outboxStore        *repository.OutboxStore
	retentionStore     *repository.RetentionStore

	staffService          *service.StaffService
	teacherProfileService *service.TeacherProfileService
	adminStaffService     *service.AdminStaffService

	staffHandler          *handler.StaffHandler
	teacherProfileHandler *handler.TeacherProfileHandler
	adminStaffHandler     *handler.AdminStaffHandler
	timeHandler           *platformHandler.TimeHandler
}

// New wires repositories, services and handlers from shared infra. Staff
// consumes no events, so it needs no RabbitMQ connection here — publishing
// goes through the outbox worker main.go starts.
func New(cfg *config.Config, pool *pgxpool.Pool) *Module {
	staffRepo := repository.NewStaffRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	teacherProfileRepo := repository.NewTeacherProfileRepository(pool)
	adminStaffRepo := repository.NewAdminStaffRepository(pool)

	staffSvc := service.NewStaffService(staffRepo)
	teacherProfileSvc := service.NewTeacherProfileService(teacherProfileRepo)
	adminStaffSvc := service.NewAdminStaffService(adminStaffRepo)

	return &Module{
		cfg:                   cfg,
		pool:                  pool,
		staffRepo:             staffRepo,
		outboxRepo:            outboxRepo,
		teacherProfileRepo:    teacherProfileRepo,
		adminStaffRepo:        adminStaffRepo,
		outboxStore:           repository.NewOutboxStore(outboxRepo),
		retentionStore:        repository.NewRetentionStore(pool),
		staffService:          staffSvc,
		teacherProfileService: teacherProfileSvc,
		adminStaffService:     adminStaffSvc,
		staffHandler:          handler.NewStaffHandler(staffSvc),
		teacherProfileHandler: handler.NewTeacherProfileHandler(teacherProfileSvc),
		adminStaffHandler:     handler.NewAdminStaffHandler(adminStaffSvc),
		timeHandler:           platformHandler.NewTimeHandler(),
	}
}

func (m *Module) Name() string { return "staff" }

// OutboxStore exposes the eventbus.OutboxStore for the outbox worker
// started in main.go.
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// RetentionStore exposes this schema's event tables to the retention worker.
func (m *Module) RetentionStore() eventbus.RetentionStore { return m.retentionStore }

// RegisterRoutes mounts /api/staff/*. The internal sub-tree catalog and
// student call lives at the root — see RegisterPublicRoutes.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	// Public profile lookup under the API prefix — no auth required so
	// anonymous visitors can browse instructor pages. Mounted before the
	// JWT-protected group so the public route wins the match.
	rg.GET("/profile/:id", m.teacherProfileHandler.GetTeacherProfileByStaffID)

	rg.Use(platformMiddleware.JWTAuth())
	rg.Use(platformMiddleware.CSRFProtection())
	rg.Use(platformMiddleware.UserRateLimit())
	{
		// A staff record carries phone and office; any teacher may read their
		// own, only admin reads anyone else's.
		rg.GET("/:id", platformMiddleware.RequireSelfOrRole("id", "admin"), m.staffHandler.GetStaffByID)

		admin := rg.Group("")
		admin.Use(platformMiddleware.RequireAdmin())
		{
			// Listing is a directory of every employee's contact details.
			// Other services reach the same reads over /internal instead.
			admin.GET("", m.staffHandler.ListStaff)
			admin.GET("/instructors", m.staffHandler.GetInstructorsByDepartment)

			admin.POST("", m.staffHandler.CreateStaff)
			admin.PUT("/:id", m.staffHandler.UpdateStaff)
			admin.DELETE("/:id", m.staffHandler.DeleteStaff)
			admin.PUT("/:id/profile", m.teacherProfileHandler.UpdateTeacherProfile)
		}

		// Time Machine admin endpoints under /api/staff/admin (kept under
		// staff for now — matches the microservice URL the frontend uses).
		timeAdmin := rg.Group("/admin")
		timeAdmin.Use(platformMiddleware.RequireAdmin())
		m.timeHandler.RegisterRoutes(timeAdmin)
	}
}

// RegisterPublicRoutes implements httpserver.PublicRoutesProvider.
func (m *Module) RegisterPublicRoutes(r *gin.Engine) {
	// Same reads as the JWT routes, reached by other services with the shared
	// secret instead of a user token. Mounted at the root rather than under
	// /api because Caddy only proxies /api — these are unreachable from
	// outside the compose network. RequireInternalSecret stays as the second
	// line: the gateway rule is one config edit away from being wrong.
	internal := r.Group("/internal")
	internal.Use(platformMiddleware.RequireInternalSecret(m.cfg.Server.InternalSecret))
	{
		internal.GET("/staff/:id", m.staffHandler.GetStaffByID)
		internal.GET("/staff", m.staffHandler.GetInstructorsByDepartment)
	}

	// The administrative staff directory. It has its own /api prefix, which
	// Caddy routes here and the SPA calls, so it is mounted at the root with
	// the same chain /api/staff gets. Every route is admin-only.
	adminStaff := r.Group("/api/admin-staff")
	adminStaff.Use(platformMiddleware.JWTAuth())
	adminStaff.Use(platformMiddleware.CSRFProtection())
	adminStaff.Use(platformMiddleware.UserRateLimit())
	adminStaff.Use(platformMiddleware.RequireAdmin())
	{
		adminStaff.GET("", m.adminStaffHandler.List)
		adminStaff.GET("/:id", m.adminStaffHandler.Get)
		adminStaff.POST("", platformMiddleware.Idempotency(), m.adminStaffHandler.Create)
		adminStaff.PUT("/:id", m.adminStaffHandler.Update)
	}
}
