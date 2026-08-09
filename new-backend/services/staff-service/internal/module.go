// Package staff wires the staff service's dependencies and exposes the
// platform-level Module + lifecycle hooks cmd/main.go consumes.
//
// The service owns the staff database (staff, outbox_events,
// teacher_profiles) and publishes staff.created/updated/deactivated events
// through its own outbox table.
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
	outboxStore        *repository.OutboxStore
	retentionStore     *repository.RetentionStore

	staffService          *service.StaffService
	teacherProfileService *service.TeacherProfileService

	staffHandler          *handler.StaffHandler
	teacherProfileHandler *handler.TeacherProfileHandler
	timeHandler           *platformHandler.TimeHandler
}

// New wires repositories, services and handlers from shared infra. Staff
// consumes no events, so it needs no RabbitMQ connection here — publishing
// goes through the outbox worker main.go starts.
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
		retentionStore:        repository.NewRetentionStore(pool),
		staffService:          staffSvc,
		teacherProfileService: teacherProfileSvc,
		staffHandler:          handler.NewStaffHandler(staffSvc),
		teacherProfileHandler: handler.NewTeacherProfileHandler(teacherProfileSvc),
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
}
