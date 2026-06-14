package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baaaki/mydreamcampus/enrollment-service/config"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/handler"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/repository"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/service"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/database"
	"github.com/baaaki/mydreamcampus/shared/audit"
	sharedHandler "github.com/baaaki/mydreamcampus/shared/handler"
	"github.com/baaaki/mydreamcampus/shared/logger"
	sharedMiddleware "github.com/baaaki/mydreamcampus/shared/middleware"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	sharedRedis "github.com/baaaki/mydreamcampus/shared/redis"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/repository"
	"github.com/baaaki/mydreamcampus/shared/semester"
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

	logger.Info("starting enrollment-service",
		zap.String("environment", cfg.Server.Environment),
		zap.String("port", cfg.Server.Port),
	)

	// Initialize database
	pool, err := database.NewPostgresPool(cfg.Database.URL)
	if err != nil {
		logger.Fatal("failed to connect to database",
			zap.Error(err),
		)
	}
	defer pool.Close()

	logger.Info("database connection established")

	// Initialize Redis
	redisClient, err := sharedRedis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		logger.Fatal("failed to connect to Redis",
			zap.Error(err),
		)
	}
	defer redisClient.Close()

	logger.Info("Redis connection established")

	// Initialize RabbitMQ
	rabbitConn, err := rabbitmq.NewConnection(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Fatal("failed to connect to RabbitMQ",
			zap.Error(err),
		)
	}
	defer rabbitConn.Close()

	logger.Info("RabbitMQ connection established")

	// Setup RabbitMQ exchanges and queues
	if err := setupRabbitMQ(rabbitConn); err != nil {
		logger.Fatal("failed to setup RabbitMQ",
			zap.Error(err),
		)
	}

	// Initialize publisher and consumer
	publisher := rabbitmq.NewPublisher(rabbitConn)
	consumer := rabbitmq.NewConsumer(rabbitConn)

	// Initialize repositories
	enrollmentRepo := repository.NewEnrollmentRepository(pool)
	courseRepo := repository.NewCourseRepository(pool)
	studentRepo := repository.NewStudentRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	processedEventsRepo := repository.NewProcessedEventsRepository(pool)

	// Initialize shared repositories
	periodRepo := sharedRepo.NewSimplePeriodRepository(pool)

	// Initialize services
	enrollmentService := service.NewEnrollmentService(
		enrollmentRepo,
		studentRepo,
		courseRepo,
		periodRepo,
	)
	eventService := service.NewEventService(
		studentRepo,
		courseRepo,
		enrollmentRepo,
		processedEventsRepo,
	)

	// Initialize handlers
	enrollmentHandler := handler.NewEnrollmentHandler(enrollmentService)

	// Initialize workers
	outboxWorker := worker.NewOutboxWorker(
		outboxRepo,
		publisher,
		5*time.Second, // Poll every 5 seconds
		10,            // Process 10 events at a time
	)

	eventConsumer := worker.NewEventConsumer(
		consumer,
		eventService,
		processedEventsRepo,
	)

	// Start workers in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start outbox worker
	go outboxWorker.Start(ctx)

	// Start event consumer
	if err := eventConsumer.Start(ctx); err != nil {
		logger.Fatal("failed to start event consumer",
			zap.Error(err),
		)
	}

	// Initialize rate limiter
	if cfg.RateLimit.Enabled {
		rlConfig := sharedMiddleware.RateLimitConfig{
			Enabled:     true,
			ServiceName: "enrollment",
			IPLimit:     cfg.RateLimit.IPLimit,
			IPWindow:   time.Duration(cfg.RateLimit.IPWindowSecs) * time.Second,
			UserLimit:  cfg.RateLimit.UserLimit,
			UserWindow: time.Duration(cfg.RateLimit.UserWindowSecs) * time.Second,
		}
		sharedMiddleware.SetRateLimiter(sharedMiddleware.NewRateLimiter(redisClient, rlConfig))
		logger.Info("rate limiter configured")
	}

	// Initialize semester checker and audit logger (via catalog service HTTP)
	semesterChecker := semester.NewHTTPChecker(cfg.CatalogService.BaseURL)
	auditLogger := audit.NewHTTPLogger(cfg.CatalogService.BaseURL, "enrollment")

	// Initialize shared handlers
	timeHandler := sharedHandler.NewTimeHandler()
	periodHandler := sharedHandler.NewSimplePeriodHandler(periodRepo, semesterChecker, auditLogger)
	internalPeriodHandler := sharedHandler.NewInternalPeriodHandler(periodRepo)

	// Setup Gin router
	router := setupRouter(enrollmentHandler, timeHandler, periodHandler, internalPeriodHandler, cfg.Server.Environment)

	// Health: liveness (process up). Ready: deps reachable.
	healthChecks := map[string]sharedHandler.HealthCheck{
		"database": pool.Ping,
		"rabbitmq": rabbitConn.Ping,
	}
	if redisClient != nil {
		healthChecks["redis"] = redisClient.Ping
	}
	router.GET("/health", sharedHandler.LivenessHandler("enrollment-service"))
	router.GET("/ready", sharedHandler.ReadinessHandler("enrollment-service", healthChecks))

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

	// Cancel workers context
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

func setupRouter(enrollmentHandler *handler.EnrollmentHandler, timeHandler *sharedHandler.TimeHandler, periodHandler *sharedHandler.SimplePeriodHandler, internalPeriodHandler *sharedHandler.InternalPeriodHandler, env string) *gin.Engine {
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

	// API routes - All routes are protected via Traefik forward-auth
	// User info is extracted from X-User-* headers set by Traefik
	api := router.Group("/api/enrollment")
	api.Use(sharedMiddleware.ExtractUserFromHeaders())
	api.Use(sharedMiddleware.CSRFProtection())
	api.Use(sharedMiddleware.UserRateLimit())
	{
		// Student routes - students can view and manage their enrollments
		student := api.Group("")
		student.Use(sharedMiddleware.RequireStudent())
		{
			student.GET("/available-courses", enrollmentHandler.GetAvailableCourses)
			student.POST("/programs", enrollmentHandler.CreateEnrollmentProgram)
			student.DELETE("/programs", enrollmentHandler.CancelMyEnrollment)
			student.GET("/my-enrollments", enrollmentHandler.GetMyEnrollments)
			student.GET("/latest-rejection", enrollmentHandler.GetLatestRejection)
			student.GET("/my-rejections", enrollmentHandler.GetMyRejections)
		}

		// Advisor routes - teachers can approve/reject enrollment programs
		advisor := api.Group("/advisor")
		advisor.Use(sharedMiddleware.RequireRole("teacher", "admin"))
		{
			advisor.GET("/pending-programs", enrollmentHandler.GetPendingProgramsByAdvisor)
			advisor.POST("/programs/:program_id/approve", enrollmentHandler.ApproveEnrollmentProgram)
			advisor.POST("/programs/:program_id/reject", enrollmentHandler.RejectEnrollmentProgram)
		}

		// Admin routes (require admin role)
		admin := api.Group("/admin")
		admin.Use(sharedMiddleware.RequireAdmin())
		{
			timeHandler.RegisterRoutes(admin)
			periodHandler.RegisterRoutes(admin)
		}
	}

	// Internal routes (service-to-service, no auth)
	internal := router.Group("/api/enrollment/internal")
	internal.Use(sharedMiddleware.StripInternalHeaders())
	{
		internalPeriodHandler.RegisterRoutes(internal)
	}

	return router
}

func setupRabbitMQ(conn *rabbitmq.Connection) error {
	channel := conn.Channel()

	// Declare enrollment exchange (for publishing enrollment events)
	if err := channel.ExchangeDeclare(
		"enrollment.events", // name
		"topic",             // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	); err != nil {
		return fmt.Errorf("failed to declare enrollment exchange: %w", err)
	}

	logger.Info("RabbitMQ exchange declared",
		zap.String("exchange", "enrollment.events"),
	)

	// Declare enrollment queue for consuming course-catalog and student events
	queueName := "enrollment.events"
	_, err := channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare enrollment queue: %w", err)
	}

	// Bind to course-catalog exchange
	if err := channel.QueueBind(
		queueName,       // queue name
		"course.#",      // routing key pattern (# matches multiple words like course.semester.created)
		"course.events", // exchange
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to bind to course-catalog exchange: %w", err)
	}

	// Bind to student exchange
	if err := channel.QueueBind(
		queueName,        // queue name
		"student.*",      // routing key pattern
		"student.events", // exchange
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to bind to student exchange: %w", err)
	}

	logger.Info("RabbitMQ queue bindings created",
		zap.String("queue", queueName),
		zap.Strings("bindings", []string{"course.events", "student.events"}),
	)

	// Pre-declare downstream consumer queues so messages persist even when consumers are offline
	publisher := rabbitmq.NewPublisher(conn)

	downstreamBindings := []struct {
		queue      string
		exchange   string
		routingKey string
	}{
		// grades-service queues
		{"grades-service-enrollment", "enrollment.events", "enrollment.program.approved"},
		// attendance-service queues
		{"attendance.events", "enrollment.events", "enrollment.program.approved"},
	}

	for _, b := range downstreamBindings {
		if err := publisher.DeclareAndBindQueue(b.queue, b.exchange, b.routingKey); err != nil {
			return fmt.Errorf("failed to declare downstream queue %s: %w", b.queue, err)
		}
	}

	logger.Info("downstream consumer queues pre-declared")

	return nil
}
