package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baaaki/mydreamcampus/auth-service/config"
	"github.com/baaaki/mydreamcampus/auth-service/internal/handler"
	"github.com/baaaki/mydreamcampus/auth-service/internal/repository"
	"github.com/baaaki/mydreamcampus/auth-service/internal/service"
	"github.com/baaaki/mydreamcampus/auth-service/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/audit"
	"github.com/baaaki/mydreamcampus/shared/database"
	sharedHandler "github.com/baaaki/mydreamcampus/shared/handler"
	"github.com/baaaki/mydreamcampus/shared/logger"
	sharedMiddleware "github.com/baaaki/mydreamcampus/shared/middleware"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	"github.com/baaaki/mydreamcampus/shared/redis"
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

	// Initialize security audit logger
	audit.InitSecurity(cfg.Server.Environment)
	defer audit.SyncSecurity()

	logger.Info("starting auth-service",
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
	redisClient, err := redis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		logger.Fatal("failed to connect to Redis",
			zap.Error(err),
		)
	}
	defer redisClient.Close()

	logger.Info("Redis connection established")

	// Set Redis client as blacklist checker for JWT middleware
	sharedMiddleware.SetBlacklistChecker(redisClient)
	logger.Info("JWT blacklist checker configured")

	// Initialize rate limiter
	if cfg.RateLimit.Enabled {
		rlConfig := sharedMiddleware.RateLimitConfig{
			Enabled:     true,
			ServiceName: "auth",
			IPLimit:  cfg.RateLimit.IPLimit,
			IPWindow: time.Duration(cfg.RateLimit.IPWindowSecs) * time.Second,
			UserLimit:  cfg.RateLimit.UserLimit,
			UserWindow: time.Duration(cfg.RateLimit.UserWindowSecs) * time.Second,
			EndpointLimits: map[string]sharedMiddleware.EndpointLimit{
				// FailClosed: bypassing rate limit on auth-critical paths
				// is unacceptable; if Redis is down we'd rather return 503
				// than allow brute force.
				"login":    {Limit: cfg.RateLimit.LoginLimit, Window: time.Duration(cfg.RateLimit.LoginWindowSecs) * time.Second, FailClosed: true},
				"refresh":  {Limit: cfg.RateLimit.RefreshLimit, Window: time.Duration(cfg.RateLimit.RefreshWindowSecs) * time.Second, FailClosed: true},
				"password": {Limit: cfg.RateLimit.PasswordLimit, Window: time.Duration(cfg.RateLimit.PasswordWindowSecs) * time.Second, FailClosed: true},
			},
		}
		rateLimiter := sharedMiddleware.NewRateLimiter(redisClient, rlConfig)
		sharedMiddleware.SetRateLimiter(rateLimiter)
		logger.Info("rate limiter configured",
			zap.Int("ip_limit", cfg.RateLimit.IPLimit),
			zap.Int("user_limit", cfg.RateLimit.UserLimit),
		)
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

	// Initialize consumer
	consumer := rabbitmq.NewConsumer(rabbitConn)

	// Initialize repositories
	authRepo := repository.NewAuthRepository(pool)
	sessionRepo := repository.NewSessionRepository(pool)
	eventRepo := repository.NewEventRepository(pool)

	// Initialize services
	authService := service.NewAuthService(authRepo, sessionRepo, eventRepo, redisClient, cfg)
	eventService := service.NewEventService(authRepo, eventRepo, pool)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, cfg)
	timeHandler := sharedHandler.NewTimeHandler()

	// Seed admin user
	ctx := context.Background()
	if err := authService.SeedAdmin(ctx); err != nil {
		logger.Error("failed to seed admin user",
			zap.Error(err),
		)
		// Don't fail on seed error, admin might already exist
	}

	// Start cleanup scheduler
	cleanupCtx, cleanupCancel := context.WithCancel(ctx)
	defer cleanupCancel()
	authService.StartCleanupScheduler(cleanupCtx)

	// Initialize event consumer
	eventConsumer := worker.NewEventConsumer(consumer, eventService)

	// Start event consumer
	consumerCtx, consumerCancel := context.WithCancel(ctx)
	defer consumerCancel()

	if err := eventConsumer.Start(consumerCtx); err != nil {
		logger.Fatal("failed to start event consumer",
			zap.Error(err),
		)
	}

	// Setup Gin router
	router := setupRouter(authHandler, timeHandler, cfg.Server.Environment)

	// Health checks: /health is liveness (process up), /ready pings deps
	router.GET("/health", sharedHandler.LivenessHandler("auth-service"))
	router.GET("/ready", sharedHandler.ReadinessHandler("auth-service", map[string]sharedHandler.HealthCheck{
		"database": pool.Ping,
		"redis":    redisClient.Ping,
		"rabbitmq": rabbitConn.Ping,
	}))

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

	// Cancel cleanup and consumer contexts
	cleanupCancel()
	consumerCancel()

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

func setupRouter(authHandler *handler.AuthHandler, timeHandler *sharedHandler.TimeHandler, env string) *gin.Engine {
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

	// API routes
	api := router.Group("/api/auth")
	{
		// Public routes (no auth required)
		api.POST("/login", sharedMiddleware.EndpointRateLimit("login"), authHandler.Login)
		api.POST("/refresh", sharedMiddleware.EndpointRateLimit("refresh"), authHandler.RefreshToken)

		// Protected routes (JWT auth required)
		protected := api.Group("")
		protected.Use(sharedMiddleware.JWTAuth())
		protected.Use(sharedMiddleware.CSRFProtection())
		protected.Use(sharedMiddleware.UserRateLimit())
		{
			protected.POST("/logout", authHandler.Logout)
			protected.POST("/logout-all", authHandler.LogoutAll)
			// Password change is sensitive: re-run JWT auth in fail-closed
			// mode so a stale/blacklisted token cannot slip through if Redis
			// is unreachable. The earlier JWTAuth() above already populated
			// claims; this layer only enforces the revocation check.
			protected.POST("/change-password",
				sharedMiddleware.JWTAuth(sharedMiddleware.WithFailClosed()),
				sharedMiddleware.EndpointRateLimit("password"),
				authHandler.ChangePassword,
			)
			protected.GET("/sessions", authHandler.GetSessions)
			protected.DELETE("/sessions/:id", authHandler.DeleteSession)
			// Traefik forward auth endpoint
			protected.GET("/verify", authHandler.Verify)
		}
	}

	// Admin routes for Time Machine (protected via Traefik forward-auth)
	admin := router.Group("/api/auth/admin")
	admin.Use(sharedMiddleware.ExtractUserFromHeaders())
	admin.Use(sharedMiddleware.RequireAdmin())
	{
		timeHandler.RegisterRoutes(admin)
	}

	return router
}

func setupRabbitMQ(conn *rabbitmq.Connection) error {
	channel := conn.Channel()

	// Declare student exchange (if not already declared by student-service)
	if err := channel.ExchangeDeclare(
		"student.events",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to declare student exchange: %w", err)
	}

	// Declare staff exchange (if not already declared by staff-service)
	if err := channel.ExchangeDeclare(
		"staff.events",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to declare staff exchange: %w", err)
	}

	// Declare auth events queue
	_, err := channel.QueueDeclare(
		"auth_events_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare auth events queue: %w", err)
	}

	// Bind queue to student exchange
	studentRoutingKeys := []string{
		"student.created",
		"student.updated",
		"student.deactivated",
	}
	for _, key := range studentRoutingKeys {
		if err := channel.QueueBind(
			"auth_events_queue",
			key,
			"student.events",
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to student exchange: %w", err)
		}
	}

	// Bind queue to staff exchange
	staffRoutingKeys := []string{
		"staff.created",
		"staff.updated",
		"staff.deactivated",
	}
	for _, key := range staffRoutingKeys {
		if err := channel.QueueBind(
			"auth_events_queue",
			key,
			"staff.events",
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to staff exchange: %w", err)
		}
	}

	logger.Info("RabbitMQ setup completed",
		zap.String("queue", "auth_events_queue"),
		zap.Strings("student_routing_keys", studentRoutingKeys),
		zap.Strings("staff_routing_keys", staffRoutingKeys),
	)

	return nil
}
