// Package staff wires the staff module's dependencies and exposes the
// platform-level Module + lifecycle hooks main.go consumes.
//
// The module owns the staff schema (staff, outbox_events, teacher_profiles)
// and publishes staff.created/updated/deactivated events through its own
// outbox table.
package staff

import (
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/handler"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/service"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	platformHandler "github.com/baaaki/mydreamcampus/shared/platform/handler"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
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
	outboxStore        *repository.OutboxStore

	staffService          *service.StaffService
	teacherProfileService *service.TeacherProfileService

	staffHandler          *handler.StaffHandler
	teacherProfileHandler *handler.TeacherProfileHandler
	timeHandler           *platformHandler.TimeHandler
}

// New wires repositories, services and handlers from shared infra.
// rabbitmq + redis are not used by staff yet (no consumer, no rate-limit
// state); they're plumbed through main.go to other modules instead.
func New(cfg *config.Config, pool *pgxpool.Pool) *Module {
	staffRepo := repository.NewStaffRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	teacherProfileRepo := repository.NewTeacherProfileRepository(pool)

	staffSvc := service.NewStaffService(staffRepo)
	teacherProfileSvc := service.NewTeacherProfileService(teacherProfileRepo)

	return &Module{
		cfg:                   cfg,
		pool:                  pool,
		staffRepo:             staffRepo,
		outboxRepo:            outboxRepo,
		teacherProfileRepo:    teacherProfileRepo,
		outboxStore:           repository.NewOutboxStore(outboxRepo),
		staffService:          staffSvc,
		teacherProfileService: teacherProfileSvc,
		staffHandler:          handler.NewStaffHandler(staffSvc),
		teacherProfileHandler: handler.NewTeacherProfileHandler(teacherProfileSvc),
		timeHandler:           platformHandler.NewTimeHandler(),
	}
}

func (m *Module) Name() string { return "staff" }

// OutboxStore exposes the eventbus.OutboxStore for the per-module outbox
// worker started in main.go.
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// StaffService is the in-process handle other modules use for staff lookups.
// When staff splits out we'll switch the
// return type to a small Service interface backed by an HTTP client.
func (m *Module) StaffService() *service.StaffService { return m.staffService }

// RegisterRoutes mounts /api/staff/*, including the /internal sub-tree
// catalog and student call over internal REST.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	// Public profile lookup under the API prefix — no auth required so
	// anonymous visitors can browse instructor pages. Mounted before the
	// JWT-protected group so the public route wins the match.
	rg.GET("/profile/:id", m.teacherProfileHandler.GetTeacherProfileByStaffID)

	// Internal sub-tree — same reads as the JWT routes above, reached by
	// other services with the shared secret instead of a user token. Mounted
	// before rg.Use(JWTAuth()) so it does not inherit the user auth chain.
	// Phase 4 moves this group out of /api, where Caddy cannot reach it.
	internal := rg.Group("/internal")
	internal.Use(platformMiddleware.RequireInternalSecret(m.cfg.Server.InternalSecret))
	{
		internal.GET("/staff/:id", m.staffHandler.GetStaffByID)
		internal.GET("/staff", m.staffHandler.GetInstructorsByDepartment)
	}

	rg.Use(platformMiddleware.JWTAuth())
	rg.Use(platformMiddleware.CSRFProtection())
	rg.Use(platformMiddleware.UserRateLimit())
	{
		rg.GET("", m.staffHandler.ListStaff)
		rg.GET("/:id", m.staffHandler.GetStaffByID)
		rg.GET("/instructors", m.staffHandler.GetInstructorsByDepartment)

		admin := rg.Group("")
		admin.Use(platformMiddleware.RequireAdmin())
		{
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
// Anonymous teacher browsing lives outside /api so the front-end can keep
// hitting /public/teachers without an auth token.
func (m *Module) RegisterPublicRoutes(r *gin.Engine) {
	public := r.Group("/public/teachers")
	public.GET("", m.teacherProfileHandler.ListTeacherProfiles)
	public.GET("/:id", m.teacherProfileHandler.GetTeacherProfileByStaffID)
}
