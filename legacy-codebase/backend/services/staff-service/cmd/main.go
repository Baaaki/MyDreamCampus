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
	"github.com/baaaki/mydreamcampus/staff-service/config"
	"github.com/baaaki/mydreamcampus/staff-service/internal/handler"
	"github.com/baaaki/mydreamcampus/staff-service/internal/repository"
	"github.com/baaaki/mydreamcampus/staff-service/internal/service"
	"github.com/baaaki/mydreamcampus/staff-service/internal/worker"
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

	logger.Info("starting staff-service",
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
				ServiceName: "staff",
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
	staffRepo := repository.NewStaffRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	teacherProfileRepo := repository.NewTeacherProfileRepository(pool)

	// Initialize services
	staffService := service.NewStaffService(staffRepo)
	teacherProfileService := service.NewTeacherProfileService(teacherProfileRepo)

	// Initialize handlers
	staffHandler := handler.NewStaffHandler(staffService)
	teacherProfileHandler := handler.NewTeacherProfileHandler(teacherProfileService)
	timeHandler := sharedHandler.NewTimeHandler()

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
	router := setupRouter(staffHandler, teacherProfileHandler, timeHandler, cfg.Server.Environment)

	// Health: liveness (process up). Ready: deps reachable.
	healthChecks := map[string]sharedHandler.HealthCheck{
		"database": pool.Ping,
		"rabbitmq": rabbitConn.Ping,
	}
	if redisClient != nil {
		healthChecks["redis"] = redisClient.Ping
	}
	router.GET("/health", sharedHandler.LivenessHandler("staff-service"))
	router.GET("/ready", sharedHandler.ReadinessHandler("staff-service", healthChecks))

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

func setupRouter(staffHandler *handler.StaffHandler, teacherProfileHandler *handler.TeacherProfileHandler, timeHandler *sharedHandler.TimeHandler, env string) *gin.Engine {
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

	// Public API routes - NO authentication required
	// Teacher profiles are public for everyone to view
	public := router.Group("/public/teachers")
	{
		public.GET("", teacherProfileHandler.ListTeacherProfiles)
		public.GET("/:id", teacherProfileHandler.GetTeacherProfileByStaffID)
	}

	// Public staff profile route - accessible without auth (for frontend to view teacher profiles)
	// This route is matched before the authenticated /api/staff group
	router.GET("/api/staff/profile/:id", teacherProfileHandler.GetTeacherProfileByStaffID)

	// Internal API routes - for service-to-service communication (no auth required)
	// These should only be accessible from internal network
	internal := router.Group("/internal/staff")
	{
		internal.GET("/:id", staffHandler.GetStaffByID)
		internal.GET("/instructors", staffHandler.GetInstructorsByDepartment)
	}

	// API routes - All routes are protected via Traefik forward-auth
	// User info is extracted from X-User-* headers set by Traefik
	api := router.Group("/api/staff")
	api.Use(sharedMiddleware.ExtractUserFromHeaders())
	api.Use(sharedMiddleware.CSRFProtection())
	api.Use(sharedMiddleware.UserRateLimit())
	{
		// Read operations - any authenticated user
		api.GET("", staffHandler.ListStaff)
		api.GET("/:id", staffHandler.GetStaffByID)
		api.GET("/instructors", staffHandler.GetInstructorsByDepartment)
		// Note: GET /api/staff/profile/:id is handled by the public route above

		// Admin only routes
		admin := api.Group("")
		admin.Use(sharedMiddleware.RequireAdmin())
		{
			admin.POST("", staffHandler.CreateStaff)
			admin.PUT("/:id", staffHandler.UpdateStaff)
			admin.DELETE("/:id", staffHandler.DeleteStaff)
			// Teacher profile update (admin only)
			admin.PUT("/:id/profile", teacherProfileHandler.UpdateTeacherProfile)
		}
	}

	// Admin routes for Time Machine
	timeAdmin := router.Group("/api/staff/admin")
	timeAdmin.Use(sharedMiddleware.ExtractUserFromHeaders())
	timeAdmin.Use(sharedMiddleware.RequireAdmin())
	{
		timeHandler.RegisterRoutes(timeAdmin)
	}

	return router
}

func setupRabbitMQ(conn *rabbitmq.Connection) error {
	channel := conn.Channel()

	// Declare staff exchange
	if err := channel.ExchangeDeclare(
		"staff.events", // name
		"topic",        // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	logger.Info("RabbitMQ exchange declared",
		zap.String("exchange", "staff.events"),
	)

	// Pre-declare downstream consumer queues so messages persist even when consumers are offline
	publisher := rabbitmq.NewPublisher(conn)

	downstreamBindings := []struct {
		queue      string
		exchange   string
		routingKey string
	}{
		// auth-service queues
		{"auth_events_queue", "staff.events", "staff.created"},
		{"auth_events_queue", "staff.events", "staff.updated"},
		{"auth_events_queue", "staff.events", "staff.deactivated"},
	}

	for _, b := range downstreamBindings {
		if err := publisher.DeclareAndBindQueue(b.queue, b.exchange, b.routingKey); err != nil {
			return fmt.Errorf("failed to declare downstream queue %s: %w", b.queue, err)
		}
	}

	logger.Info("downstream consumer queues pre-declared")

	return nil
}
