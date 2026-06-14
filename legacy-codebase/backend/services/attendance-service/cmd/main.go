package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baaaki/mydreamcampus/attendance-service/config"
	"github.com/baaaki/mydreamcampus/attendance-service/internal/handler"
	"github.com/baaaki/mydreamcampus/attendance-service/internal/repository"
	"github.com/baaaki/mydreamcampus/attendance-service/internal/service"
	"github.com/baaaki/mydreamcampus/attendance-service/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/audit"
	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/database"
	"github.com/baaaki/mydreamcampus/shared/events"
	sharedHandler "github.com/baaaki/mydreamcampus/shared/handler"
	"github.com/baaaki/mydreamcampus/shared/logger"
	sharedMiddleware "github.com/baaaki/mydreamcampus/shared/middleware"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	sharedRedis "github.com/baaaki/mydreamcampus/shared/redis"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/repository"
	"github.com/baaaki/mydreamcampus/shared/semester"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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

	logger.Info("starting attendance-service",
		zap.String("environment", cfg.Server.Environment),
		zap.String("port", cfg.Server.Port),
	)

	// Initialize database
	pool, err := database.NewPostgresPool(cfg.Database.URL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	logger.Info("database connection established")

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}

	logger.Info("Redis connection established")

	// Initialize shared Redis client for rate limiting
	sharedRedisClient, err := sharedRedis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		logger.Warn("shared Redis not available, rate limiting disabled", zap.Error(err))
	} else {
		defer sharedRedisClient.Close()
		if cfg.RateLimit.Enabled {
			rlConfig := sharedMiddleware.RateLimitConfig{
				Enabled:     true,
				ServiceName: "attendance",
				IPLimit:     cfg.RateLimit.IPLimit,
				IPWindow:   time.Duration(cfg.RateLimit.IPWindowSecs) * time.Second,
				UserLimit:  cfg.RateLimit.UserLimit,
				UserWindow: time.Duration(cfg.RateLimit.UserWindowSecs) * time.Second,
			}
			sharedMiddleware.SetRateLimiter(sharedMiddleware.NewRateLimiter(sharedRedisClient, rlConfig))
			logger.Info("rate limiter configured")
		}
	}

	// Initialize RabbitMQ
	rabbitConn, err := rabbitmq.NewConnection(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Fatal("failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rabbitConn.Close()

	logger.Info("RabbitMQ connection established")

	// Setup RabbitMQ
	if err := setupRabbitMQ(rabbitConn); err != nil {
		logger.Fatal("failed to setup RabbitMQ", zap.Error(err))
	}

	// Initialize publisher and consumer
	publisher := rabbitmq.NewPublisher(rabbitConn)
	consumer := rabbitmq.NewConsumer(rabbitConn)

	// Initialize repositories
	cacheRepo := repository.NewCacheRepository(pool)
	sessionRepo := repository.NewSessionRepository(pool)
	attendanceRepo := repository.NewAttendanceRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	eventRepo := repository.NewEventRepository(pool)
	periodRepo := sharedRepo.NewSimplePeriodRepository(pool)

	// Initialize semester checker, semester client and audit logger (via catalog service HTTP)
	semesterChecker := semester.NewHTTPChecker(cfg.CatalogService.BaseURL)
	semesterClient := client.NewSemesterClient(cfg.CatalogService.BaseURL)
	auditLogger := audit.NewHTTPLogger(cfg.CatalogService.BaseURL, "attendance")

	// Initialize services
	qrService := service.NewQRService()
	redisService := service.NewRedisService(redisClient)
	attendanceService := service.NewAttendanceService(
		cacheRepo,
		sessionRepo,
		attendanceRepo,
		outboxRepo,
		qrService,
		redisService,
		semesterClient,
		periodRepo,
	)

	// Initialize handlers
	attendanceHandler := handler.NewAttendanceHandler(attendanceService)
	timeHandler := sharedHandler.NewTimeHandler()
	periodHandler := sharedHandler.NewSimplePeriodHandler(periodRepo, semesterChecker, auditLogger)
	internalPeriodHandler := sharedHandler.NewInternalPeriodHandler(periodRepo)

	// Initialize workers
	outboxWorker := worker.NewOutboxWorker(outboxRepo, publisher)
	eventConsumer := worker.NewEventConsumer(consumer, cacheRepo, eventRepo)
	bufferFlusher := worker.NewBufferFlusher(attendanceRepo, redisService)
	sessionExpiryHandler := worker.NewSessionExpiryHandler(sessionRepo, redisService)

	// Start workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go outboxWorker.Start(ctx)
	go eventConsumer.Start(ctx)
	go bufferFlusher.Start(ctx)
	go sessionExpiryHandler.Start(ctx)

	// Setup HTTP server
	router := setupRouter(cfg, attendanceHandler, timeHandler, periodHandler, internalPeriodHandler)

	// Health: liveness (process up). Ready: deps reachable.
	healthChecks := map[string]sharedHandler.HealthCheck{
		"database": pool.Ping,
		"rabbitmq": rabbitConn.Ping,
		"redis": func(ctx context.Context) error {
			return redisClient.Ping(ctx).Err()
		},
	}
	router.GET("/health", sharedHandler.LivenessHandler("attendance-service"))
	router.GET("/ready", sharedHandler.ReadinessHandler("attendance-service", healthChecks))

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server
	go func() {
		logger.Info("server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited")
}

func setupRouter(cfg *config.Config, attendanceHandler *handler.AttendanceHandler, timeHandler *sharedHandler.TimeHandler, periodHandler *sharedHandler.SimplePeriodHandler, internalPeriodHandler *sharedHandler.InternalPeriodHandler) *gin.Engine {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(sharedMiddleware.RequestLogger())
	router.Use(sharedMiddleware.CORS())
	router.Use(sharedMiddleware.IPRateLimit())
	router.Use(sharedMiddleware.SetCSRFToken(cfg.Server.Environment == "production"))

	// Health endpoints registered in main() with dependency checks

	// API routes - All routes are protected via Traefik forward-auth
	// User info is extracted from X-User-* headers set by Traefik
	api := router.Group("/api/attendance")
	api.Use(sharedMiddleware.ExtractUserFromHeaders())
	api.Use(sharedMiddleware.CSRFProtection())
	api.Use(sharedMiddleware.UserRateLimit())

	// Student routes
	api.POST("/scan", sharedMiddleware.RequireRole("student"), attendanceHandler.ScanQR)
	api.GET("/my", sharedMiddleware.RequireRole("student"), attendanceHandler.GetMyAttendance)

	// Instructor routes
	api.POST("/sessions", sharedMiddleware.RequireRole("teacher"), attendanceHandler.CreateSession)
	api.GET("/sessions/:session_id", sharedMiddleware.RequireRole("teacher"), attendanceHandler.GetSessionDetails)
	api.GET("/sessions/:session_id/qr", sharedMiddleware.RequireRole("teacher"), attendanceHandler.GetQRCode)
	api.GET("/sessions/:session_id/records", sharedMiddleware.RequireRole("teacher"), attendanceHandler.GetSessionRecords)
	api.GET("/sessions/:session_id/students", sharedMiddleware.RequireRole("teacher"), attendanceHandler.GetSessionStudents)
	api.POST("/sessions/:session_id/manual", sharedMiddleware.RequireRole("teacher"), attendanceHandler.CreateManualAttendance)
	api.POST("/sessions/:session_id/close", sharedMiddleware.RequireRole("teacher"), attendanceHandler.CloseSession)
	api.POST("/courses/:course_id/finalize", sharedMiddleware.RequireRole("teacher"), attendanceHandler.FinalizeAttendance)

	// Admin routes
	admin := router.Group("/api/attendance/admin")
	admin.Use(sharedMiddleware.ExtractUserFromHeaders())
	admin.Use(sharedMiddleware.RequireAdmin())
	{
		admin.GET("/sessions", attendanceHandler.AdminListSessions)
		timeHandler.RegisterRoutes(admin)
		periodHandler.RegisterRoutes(admin)
	}

	// Internal routes (service-to-service, no auth)
	internal := router.Group("/api/attendance/internal")
	internal.Use(sharedMiddleware.StripInternalHeaders())
	{
		internalPeriodHandler.RegisterRoutes(internal)
	}

	return router
}

func setupRabbitMQ(conn *rabbitmq.Connection) error {
	ch := conn.Channel()
	// Channel() returns non-nil pointer, no error

	// Declare exchanges
	exchanges := []string{"student.events", "course.events", "enrollment.events", "attendance.events"}
	for _, exchange := range exchanges {
		if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
			return err
		}
	}

	// Declare attendance.events queue for consuming events
	_, err := ch.QueueDeclare(
		"attendance.events", // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return err
	}

	// Bind queue to exchanges with relevant routing keys
	bindings := []struct {
		queue      string
		exchange   string
		routingKey string
	}{
		{"attendance.events", "student.events", "student.#"},
		{"attendance.events", "course.events", "course.semester.#"},
		{"attendance.events", "enrollment.events", "enrollment.program.approved"},
	}

	for _, b := range bindings {
		if err := ch.QueueBind(b.queue, b.routingKey, b.exchange, false, nil); err != nil {
			return err
		}
	}

	// Pre-declare downstream consumer queues so messages persist even when consumers are offline
	publisher := rabbitmq.NewPublisher(conn)

	if err := publisher.DeclareAndBindQueue("grades-service-attendance", "attendance.events", events.EventAttendanceSemesterFailed); err != nil {
		return fmt.Errorf("failed to declare downstream queue: %w", err)
	}

	logger.Info("downstream consumer queues pre-declared")

	return nil
}
