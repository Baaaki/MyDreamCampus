// Package attendance wires the attendance module's dependencies
// and exposes the platform-level Module + lifecycle hooks main.go uses.
package attendance

import (
	"github.com/baaaki/mydreamcampus/monolith/config"
	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/handler"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/service"
	ccService "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
	platformMiddleware "github.com/baaaki/mydreamcampus/monolith/internal/platform/middleware"
	platformRepo "github.com/baaaki/mydreamcampus/monolith/internal/platform/repository"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Module struct {
	cfg  *config.Config
	pool *pgxpool.Pool

	cacheRepo      *repository.CacheRepository
	sessionRepo    *repository.SessionRepository
	attendanceRepo *repository.AttendanceRepository
	outboxRepo     *repository.OutboxRepository
	outboxStore    *repository.OutboxStore

	qrService      *service.QRService
	redisService   *service.RedisService
	semesterClient service.SemesterClient

	attendanceService *service.AttendanceService
	attendanceHandler *handler.AttendanceHandler
}

func New(
	cfg *config.Config,
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	semesterSvc *ccService.SemesterService,
	periodRepo *platformRepo.SimplePeriodRepository,
) *Module {
	cacheRepo := repository.NewCacheRepository(pool)
	sessionRepo := repository.NewSessionRepository(pool)
	attendanceRepo := repository.NewAttendanceRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)

	qrService := service.NewQRService()
	redisService := service.NewRedisService(redisClient)
	semesterClient := service.NewInProcessSemesterClient(semesterSvc)

	attendanceSvc := service.NewAttendanceService(
		cacheRepo, sessionRepo, attendanceRepo, outboxRepo,
		qrService, redisService, semesterClient, periodRepo,
	)

	return &Module{
		cfg:               cfg,
		pool:              pool,
		cacheRepo:         cacheRepo,
		sessionRepo:       sessionRepo,
		attendanceRepo:    attendanceRepo,
		outboxRepo:        outboxRepo,
		outboxStore:       repository.NewOutboxStore(outboxRepo),
		qrService:         qrService,
		redisService:      redisService,
		semesterClient:    semesterClient,
		attendanceService: attendanceSvc,
		attendanceHandler: handler.NewAttendanceHandler(attendanceSvc),
	}
}

func (m *Module) Name() string { return "v1/attendance" }

func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

func (m *Module) AttendanceService() *service.AttendanceService { return m.attendanceService }

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	protected := rg.Group("")
	protected.Use(platformMiddleware.JWTAuth())
	protected.Use(platformMiddleware.CSRFProtection())
	protected.Use(platformMiddleware.UserRateLimit())
	{
		// Teacher routes
		protected.POST("/sessions", platformMiddleware.RequireRole("teacher", "admin"), m.attendanceHandler.CreateSession)
		protected.GET("/sessions/:session_id", platformMiddleware.RequireRole("teacher", "admin"), m.attendanceHandler.GetSessionDetails)
		protected.GET("/sessions/:session_id/records", platformMiddleware.RequireRole("teacher", "admin"), m.attendanceHandler.GetSessionRecords)
		protected.GET("/sessions/:session_id/students", platformMiddleware.RequireRole("teacher", "admin"), m.attendanceHandler.GetSessionStudents)
		protected.POST("/sessions/:session_id/manual", platformMiddleware.RequireRole("teacher", "admin"), m.attendanceHandler.CreateManualAttendance)
		protected.POST("/sessions/:session_id/close", platformMiddleware.RequireRole("teacher", "admin"), m.attendanceHandler.CloseSession)
		protected.GET("/sessions/:session_id/qr", platformMiddleware.RequireRole("teacher", "admin"), m.attendanceHandler.GetQRCode)

		protected.POST("/courses/:course_id/finalize", platformMiddleware.RequireRole("teacher", "admin"), m.attendanceHandler.FinalizeAttendance)

		// Student routes
		protected.POST("/scan", platformMiddleware.RequireRole("student"), m.attendanceHandler.ScanQR)
		protected.GET("/my", platformMiddleware.RequireRole("student"), m.attendanceHandler.GetMyAttendance)
		
		// Admin
		protected.GET("/admin/sessions", platformMiddleware.RequireAdmin(), m.attendanceHandler.AdminListSessions)
	}
}
