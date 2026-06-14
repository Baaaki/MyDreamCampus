package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baaaki/mydreamcampus/shared/database"
	sharedHandler "github.com/baaaki/mydreamcampus/shared/handler"
	"github.com/baaaki/mydreamcampus/shared/logger"
	sharedMiddleware "github.com/baaaki/mydreamcampus/shared/middleware"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	sharedRedis "github.com/baaaki/mydreamcampus/shared/redis"
	"github.com/baaaki/mydreamcampus/student-service/config"
	"github.com/baaaki/mydreamcampus/student-service/internal/handler"
	"github.com/baaaki/mydreamcampus/student-service/internal/repository"
	"github.com/baaaki/mydreamcampus/student-service/internal/service"
	"github.com/baaaki/mydreamcampus/student-service/internal/worker"
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

	logger.Info("starting student-service",
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

	// Initialize Redis for rate limiting
	redisClient, err := sharedRedis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		logger.Warn("Redis not available, rate limiting disabled", zap.Error(err))
	} else {
		defer redisClient.Close()
		if cfg.RateLimit.Enabled {
			rlConfig := sharedMiddleware.RateLimitConfig{
				Enabled:     true,
				ServiceName: "student",
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
	studentRepo := repository.NewStudentRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	processedEventsRepo := repository.NewProcessedEventsRepository(pool)
	importRepo := repository.NewImportRepository(pool)

	// Initialize Staff Service client
	staffClient := service.NewStaffClient(cfg.Services.StaffServiceURL)

	// Initialize services
	studentService := service.NewStudentService(studentRepo, staffClient)
	importService := service.NewImportService(importRepo, studentRepo, staffClient)

	// Initialize handlers
	studentHandler := handler.NewStudentHandler(studentService, importService)
	timeHandler := sharedHandler.NewTimeHandler()

	// Initialize workers
	outboxWorker := worker.NewOutboxWorker(
		outboxRepo,
		publisher,
		5*time.Second, // Poll every 5 seconds
		10,            // Process 10 events at a time
	)

	eventConsumer := worker.NewEventConsumer(
		consumer,
		studentRepo,
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

	// Setup Gin router
	router := setupRouter(studentHandler, timeHandler, cfg.Server.Environment)

	// Health: liveness (process up). Ready: deps reachable.
	healthChecks := map[string]sharedHandler.HealthCheck{
		"database": pool.Ping,
		"rabbitmq": rabbitConn.Ping,
	}
	if redisClient != nil {
		healthChecks["redis"] = redisClient.Ping
	}
	router.GET("/health", sharedHandler.LivenessHandler("student-service"))
	router.GET("/ready", sharedHandler.ReadinessHandler("student-service", healthChecks))

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

func setupRouter(studentHandler *handler.StudentHandler, timeHandler *sharedHandler.TimeHandler, env string) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Disable trailing slash redirect - accept both /api/students and /api/students/
	router.RedirectTrailingSlash = false

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
	api := router.Group("/api/students")
	api.Use(sharedMiddleware.ExtractUserFromHeaders())
	api.Use(sharedMiddleware.CSRFProtection())
	api.Use(sharedMiddleware.UserRateLimit())
	{
		// Read operations - any authenticated user
		api.GET("", studentHandler.ListStudents)
		api.GET("/:id", studentHandler.GetStudentByID)
		api.POST("/search", studentHandler.SearchStudents)

		// Advisor routes - teachers can view their advisees
		api.GET("/my-advisees", sharedMiddleware.RequireRole("teacher", "admin"), studentHandler.GetMyAdvisees)

		// Admin only routes
		admin := api.Group("")
		admin.Use(sharedMiddleware.RequireAdmin())
		{
			admin.POST("", studentHandler.CreateStudent)
			admin.PUT("/:id", studentHandler.UpdateStudent)
			admin.DELETE("/:id", studentHandler.DeleteStudent)
			admin.GET("/orphaned", studentHandler.ListOrphanedStudents)
			admin.PUT("/bulk-advisor-assign", studentHandler.BulkAssignAdvisor)
			admin.POST("/bulk-import", studentHandler.BulkImport)
			admin.GET("/bulk-import/:job_id", studentHandler.GetImportJobStatus)
			admin.GET("/bulk-import", studentHandler.ListImportJobs)
		}
	}

	// Admin routes for Time Machine
	timeAdmin := router.Group("/api/students/admin")
	timeAdmin.Use(sharedMiddleware.ExtractUserFromHeaders())
	timeAdmin.Use(sharedMiddleware.RequireAdmin())
	{
		timeHandler.RegisterRoutes(timeAdmin)
	}

	return router
}

func setupRabbitMQ(conn *rabbitmq.Connection) error {
	channel := conn.Channel()

	// Declare student exchange (for publishing student events)
	if err := channel.ExchangeDeclare(
		"student.events", // name
		"topic",          // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	); err != nil {
		return fmt.Errorf("failed to declare student exchange: %w", err)
	}

	logger.Info("RabbitMQ exchange declared",
		zap.String("exchange", "student.events"),
	)

	// Setup staff events queue (for consuming staff.deactivated events)
	if err := worker.SetupStaffEventsQueue(conn); err != nil {
		return fmt.Errorf("failed to setup staff events queue: %w", err)
	}

	// Pre-declare downstream consumer queues so messages persist even when consumers are offline
	publisher := rabbitmq.NewPublisher(conn)

	downstreamBindings := []struct {
		queue      string
		exchange   string
		routingKey string
	}{
		// auth-service queues
		{"auth_events_queue", "student.events", "student.created"},
		{"auth_events_queue", "student.events", "student.updated"},
		{"auth_events_queue", "student.events", "student.deactivated"},
		// enrollment-service queues
		{"enrollment.events", "student.events", "student.*"},
		// attendance-service queues
		{"attendance.events", "student.events", "student.#"},
		// grades-service queues
		{"grades-service-student", "student.events", "student.created"},
		{"grades-service-student", "student.events", "student.updated"},
		{"grades-service-student", "student.events", "student.deactivated"},
	}

	for _, b := range downstreamBindings {
		if err := publisher.DeclareAndBindQueue(b.queue, b.exchange, b.routingKey); err != nil {
			return fmt.Errorf("failed to declare downstream queue %s: %w", b.queue, err)
		}
	}

	logger.Info("downstream consumer queues pre-declared")

	return nil
}
