// Package staff wires the staff module's dependencies and exposes the
// platform-level Module + lifecycle hooks main.go consumes.
//
// The module owns the staff schema (staff, outbox_events, teacher_profiles)
// and publishes staff.created/updated/deactivated events through its own
// outbox table — see plan section 5.9 for the event catalogue.
package staff

import (
	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	platformHandler "github.com/baaaki/mydreamcampus/monolith/internal/platform/handler"
	platformMiddleware "github.com/baaaki/mydreamcampus/monolith/internal/platform/middleware"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/handler"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Module is the staff module's wiring root.
type Module struct {
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
func New(pool *pgxpool.Pool) *Module {
	staffRepo := repository.NewStaffRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	teacherProfileRepo := repository.NewTeacherProfileRepository(pool)

	staffSvc := service.NewStaffService(staffRepo)
	teacherProfileSvc := service.NewTeacherProfileService(teacherProfileRepo)

	return &Module{
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

// StaffService is the in-process handle other modules use for staff lookups
// (plan section 8 strategy 1). When staff splits out we'll switch the
// return type to a small Service interface backed by an HTTP client.
func (m *Module) StaffService() *service.StaffService { return m.staffService }

// RegisterRoutes mounts /api/staff/*. Internal service-to-service routes
// the original microservice exposed under /internal/staff/* are gone —
// other modules now reach the staff service via the in-process Service
// interface (plan section 8 strategy 1).
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	// Public profile lookup under the API prefix — no auth required so
	// anonymous visitors can browse instructor pages. Mounted before the
	// JWT-protected group so the public route wins the match.
	rg.GET("/profile/:id", m.teacherProfileHandler.GetTeacherProfileByStaffID)

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

// RegisterPublicRoutes implements monolithHTTP.PublicRoutesProvider.
// Anonymous teacher browsing lives outside /api so the front-end can keep
// hitting /public/teachers without an auth token.
func (m *Module) RegisterPublicRoutes(r *gin.Engine) {
	public := r.Group("/public/teachers")
	public.GET("", m.teacherProfileHandler.ListTeacherProfiles)
	public.GET("/:id", m.teacherProfileHandler.GetTeacherProfileByStaffID)
}
