package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baaaki/mydreamcampus/course-catalog-service/config"
	internaldb "github.com/baaaki/mydreamcampus/course-catalog-service/internal/database"
	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/handler"
	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/repository"
	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/service"
	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/worker"
	sharedHandler "github.com/baaaki/mydreamcampus/shared/handler"
	"github.com/baaaki/mydreamcampus/shared/logger"
	sharedMiddleware "github.com/baaaki/mydreamcampus/shared/middleware"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	sharedRedis "github.com/baaaki/mydreamcampus/shared/redis"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Initialize logger
	if err := logger.Init(cfg.Server.Environment); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("starting course-catalog-service",
		zap.String("environment", cfg.Server.Environment),
		zap.String("port", cfg.Server.Port),
	)

	// Initialize database with custom enum types
	pool, err := internaldb.NewPoolWithEnums(cfg.Database.URL)
	if err != nil {
		logger.Fatal("failed to connect to database",
			zap.Error(err),
		)
	}
	defer pool.Close()

	logger.Info("database connection established")

	// Initialize Redis for rate limiting
	redisClient, err := sharedRedis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		logger.Warn("Redis not available, rate limiting disabled", zap.Error(err))
	} else {
		defer redisClient.Close()
		if cfg.RateLimit.Enabled {
			rlConfig := sharedMiddleware.RateLimitConfig{
				Enabled:     true,
				ServiceName: "catalog",
				IPLimit:     cfg.RateLimit.IPLimit,
				IPWindow:   time.Duration(cfg.RateLimit.IPWindowSecs) * time.Second,
				UserLimit:  cfg.RateLimit.UserLimit,
				UserWindow: time.Duration(cfg.RateLimit.UserWindowSecs) * time.Second,
			}
			sharedMiddleware.SetRateLimiter(sharedMiddleware.NewRateLimiter(redisClient, rlConfig))
			logger.Info("rate limiter configured")
		}
	}

	// Initialize RabbitMQ
	rabbitConn, err := rabbitmq.NewConnection(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Fatal("failed to connect to RabbitMQ",
			zap.Error(err),
		)
	}
	defer rabbitConn.Close()

	logger.Info("RabbitMQ connection established")

	// Setup RabbitMQ exchange and queue
	if err := setupRabbitMQ(rabbitConn); err != nil {
		logger.Fatal("failed to setup RabbitMQ",
			zap.Error(err),
		)
	}

	// Initialize publisher
	publisher := rabbitmq.NewPublisher(rabbitConn)

	// Initialize repositories
	catalogRepo := repository.NewCatalogRepository(pool)
	semesterRepo := repository.NewSemesterRepository(pool)
	scheduleRepo := repository.NewScheduleRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)

	// Initialize audit logger (direct DB writer for catalog service)
	catalogAuditLogger := service.NewDirectAuditLogger(auditRepo, "catalog")

	// Initialize semester status repo (needs audit logger for auto-complete logging)
	semesterStatusRepo := repository.NewSemesterStatusRepository(pool, catalogAuditLogger)

	// Initialize shared repositories
	periodRepo := sharedRepo.NewSimplePeriodRepository(pool)

	// Initialize shared handlers
	timeHandler := sharedHandler.NewTimeHandler()
	periodHandler := sharedHandler.NewSimplePeriodHandler(periodRepo, semesterStatusRepo, catalogAuditLogger)

	// Initialize semester status handler
	semesterStatusHandler := handler.NewSemesterStatusHandler(semesterStatusRepo, periodRepo, catalogAuditLogger, handler.ServiceURLs{
		Enrollment: cfg.EnrollmentService.BaseURL,
		Grades:     cfg.GradesService.BaseURL,
		Attendance: cfg.AttendanceService.BaseURL,
		Meal:       cfg.MealService.BaseURL,
	}, pool, cfg.InternalServiceSecret)

	// Initialize audit handler
	auditHandler := handler.NewAuditHandler(auditRepo)

	// Initialize staff client
	staffClient := service.NewHTTPStaffClient(cfg.StaffService.BaseURL)

	// Initialize services
	catalogService := service.NewCatalogService(catalogRepo)
	semesterService := service.NewSemesterService(
		catalogRepo,
		semesterRepo,
		scheduleRepo,
		outboxRepo,
		staffClient,
		periodRepo,
		semesterStatusRepo,
	)

	// Initialize handlers
	catalogHandler := handler.NewCatalogHandler(catalogService)
	semesterHandler := handler.NewSemesterHandler(semesterService)

	// Initialize outbox worker
	outboxWorker := worker.NewOutboxWorker(
		outboxRepo,
		publisher,
		5*time.Second, // Poll every 5 seconds
		10,            // Process 10 events at a time
	)

	// Start outbox worker in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go outboxWorker.Start(ctx)

	// Setup Gin router
	router := setupRouter(catalogHandler, semesterHandler, timeHandler, periodHandler, semesterStatusHandler, auditHandler, cfg.Server.Environment)

	// Health: liveness (process up). Ready: deps reachable.
	healthChecks := map[string]sharedHandler.HealthCheck{
		"database": pool.Ping,
		"rabbitmq": rabbitConn.Ping,
	}
	if redisClient != nil {
		healthChecks["redis"] = redisClient.Ping
	}
	router.GET("/health", sharedHandler.LivenessHandler("course-catalog-service"))
	router.GET("/ready", sharedHandler.ReadinessHandler("course-catalog-service", healthChecks))

	// Start HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		logger.Info("server starting",
			zap.String("port", cfg.Server.Port),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server",
				zap.Error(err),
			)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Cancel outbox worker context
	cancel()

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown",
			zap.Error(err),
		)
	}

	logger.Info("server exited")
}

func setupRouter(catalogHandler *handler.CatalogHandler, semesterHandler *handler.SemesterHandler, timeHandler *sharedHandler.TimeHandler, periodHandler *sharedHandler.SimplePeriodHandler, semesterStatusHandler *handler.SemesterStatusHandler, auditHandler *handler.AuditHandler, env string) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(sharedMiddleware.Recovery())
	router.Use(sharedMiddleware.SecurityHeaders())
	router.Use(sharedMiddleware.CORS())
	router.Use(sharedMiddleware.RequestLogger())
	router.Use(sharedMiddleware.IPRateLimit())
	router.Use(sharedMiddleware.SetCSRFToken(env == "production"))

	// Health endpoints registered in main() with dependency checks

	// API routes
	api := router.Group("/api")
	{
		// ===== PUBLIC ROUTES (No authentication required) =====
		// These routes are accessible without Traefik forward-auth
		catalog := api.Group("/catalog")
		{
			// Public read operations - no auth required
			catalog.GET("/courses", catalogHandler.ListCourses)
			catalog.GET("/courses/:course_code", catalogHandler.GetCourseByCourseCode)
		}

		// ===== PROTECTED ROUTES (Authentication required via Traefik) =====
		// User info is extracted from X-User-* headers set by Traefik forward-auth
		protectedApi := api.Group("")
		protectedApi.Use(sharedMiddleware.ExtractUserFromHeaders())
		protectedApi.Use(sharedMiddleware.CSRFProtection())
		protectedApi.Use(sharedMiddleware.UserRateLimit())
		{
			// Catalog admin routes
			catalogAdmin := protectedApi.Group("/catalog")
			{
				catalogAdmin.POST("/courses", sharedMiddleware.RequireAdmin(), catalogHandler.CreateCourse)
				catalogAdmin.PUT("/courses/:course_code", sharedMiddleware.RequireAdmin(), catalogHandler.UpdateCourse)
			}

			// Admin routes for Time Machine, Academic Periods, Semester Status & Audit Log
			admin := protectedApi.Group("/catalog/admin")
			admin.Use(sharedMiddleware.RequireAdmin())
			{
				timeHandler.RegisterRoutes(admin)
				periodHandler.RegisterRoutes(admin)
				semesterStatusHandler.RegisterRoutes(admin)
				auditHandler.RegisterAdminRoutes(admin)
			}

			// Semester routes - all require authentication
			semesters := protectedApi.Group("/semesters")
			{
				// Teacher routes
				semesters.GET("/teacher/courses", sharedMiddleware.RequireRole("teacher"), semesterHandler.GetTeacherCourses)

				// Semester course routes
				semesterCourses := semesters.Group("/:semester_id/courses")
				{
					// Read operations - any authenticated user
					semesterCourses.GET("", semesterHandler.ListSemesterCourses)
					semesterCourses.GET("/:course_id", semesterHandler.GetSemesterCourseByID)

					// Admin only routes
					semesterCourses.POST("", sharedMiddleware.RequireAdmin(), semesterHandler.CreateSemesterCourse)
					semesterCourses.DELETE("/:course_id", sharedMiddleware.RequireAdmin(), semesterHandler.DeleteSemesterCourse)
				}
			}
		}

		// ===== INTERNAL ROUTES (Service-to-service, no auth) =====
		internal := api.Group("/catalog/internal")
		{
			semesterStatusHandler.RegisterInternalRoutes(internal)
			auditHandler.RegisterInternalRoutes(internal)
		}
	}

	return router
}

func setupRabbitMQ(conn *rabbitmq.Connection) error {
	channel := conn.Channel()

	// Declare course exchange
	if err := channel.ExchangeDeclare(
		"course.events", // name
		"topic",         // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	logger.Info("RabbitMQ exchange declared",
		zap.String("exchange", "course.events"),
	)

	// Pre-declare downstream consumer queues so messages persist even when consumers are offline
	publisher := rabbitmq.NewPublisher(conn)

	downstreamBindings := []struct {
		queue      string
		exchange   string
		routingKey string
	}{
		// enrollment-service queues
		{"enrollment.events", "course.events", "course.#"},
		// grades-service queues
		{"grades-service-course", "course.events", "course.semester.created"},
		{"grades-service-course", "course.events", "course.semester.updated"},
		{"grades-service-course", "course.events", "course.semester.deleted"},
		{"grades-service-course", "course.events", "course.instructor.changed"},
		{"grades-service-course", "course.events", "course.prerequisites.updated"},
		// attendance-service queues
		{"attendance.events", "course.events", "course.semester.#"},
	}

	for _, b := range downstreamBindings {
		if err := publisher.DeclareAndBindQueue(b.queue, b.exchange, b.routingKey); err != nil {
			return fmt.Errorf("failed to declare downstream queue %s: %w", b.queue, err)
		}
	}

	logger.Info("downstream consumer queues pre-declared")

	return nil
}
