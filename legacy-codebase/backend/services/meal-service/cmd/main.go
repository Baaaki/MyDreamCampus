package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baaaki/mydreamcampus/meal-service/config"
	"github.com/baaaki/mydreamcampus/meal-service/internal/handler"
	"github.com/baaaki/mydreamcampus/meal-service/internal/repository"
	"github.com/baaaki/mydreamcampus/meal-service/internal/service"
	"github.com/baaaki/mydreamcampus/meal-service/internal/worker"
	sharedDB "github.com/baaaki/mydreamcampus/shared/database"
	"github.com/baaaki/mydreamcampus/shared/audit"
	sharedHandler "github.com/baaaki/mydreamcampus/shared/handler"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/middleware"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
	sharedRedis "github.com/baaaki/mydreamcampus/shared/redis"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	logger.MustInit(cfg.Server.Environment)
	log := logger.Log
	defer logger.Sync()

	log.Info("starting meal service",
		zap.String("environment", cfg.Server.Environment),
		zap.String("port", cfg.Server.Port),
	)

	// Initialize database
	ctx := context.Background()
	dbPool, err := sharedDB.NewPostgresPool(cfg.Database.URL)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	log.Info("connected to database")

	// Initialize Redis for rate limiting
	redisClient, err := sharedRedis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Warn("Redis not available, rate limiting disabled", zap.Error(err))
	} else {
		defer redisClient.Close()
		if cfg.RateLimit.Enabled {
			rlConfig := middleware.RateLimitConfig{
				Enabled:     true,
				ServiceName: "meal",
				IPLimit:     cfg.RateLimit.IPLimit,
				IPWindow:   time.Duration(cfg.RateLimit.IPWindowSecs) * time.Second,
				UserLimit:  cfg.RateLimit.UserLimit,
				UserWindow: time.Duration(cfg.RateLimit.UserWindowSecs) * time.Second,
			}
			middleware.SetRateLimiter(middleware.NewRateLimiter(redisClient, rlConfig))
			log.Info("rate limiter configured")
		}
	}

	// Initialize repositories
	cafeteriaRepo := repository.NewCafeteriaRepository(dbPool)
	reservationRepo := repository.NewReservationRepository(dbPool)
	studentCacheRepo := repository.NewStudentCacheRepository(dbPool)
	menuRepo := repository.NewMenuRepository(dbPool)
	outboxRepo := repository.NewOutboxRepository(dbPool)
	processedEventsRepo := repository.NewProcessedEventsRepository(dbPool)

	// Initialize RabbitMQ
	rabbitConn, err := rabbitmq.NewConnection(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatal("failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rabbitConn.Close()

	log.Info("connected to RabbitMQ")

	// Initialize publisher
	publisher := rabbitmq.NewPublisher(rabbitConn)

	// Initialize payment client (gRPC)
	paymentClient, err := service.NewPaymentClient(cfg.Payment.GRPCAddress, log)
	if err != nil {
		log.Fatal("failed to connect to payment service", zap.Error(err))
	}
	defer paymentClient.Close()

	// Initialize closed days repository wrapped with an in-memory TTL cache.
	// Closed days rarely change; the reservation hot path answers from memory.
	closedDaysRepo := repository.NewClosedDaysCache(
		repository.NewClosedDaysRepository(dbPool),
		5*time.Minute,
	)

	// Initialize audit logger (via catalog service HTTP)
	auditLogger := audit.NewHTTPLogger(cfg.CatalogService.BaseURL, "meal")

	// Initialize shared handlers
	timeHandler := sharedHandler.NewTimeHandler()
	closedDaysHandler := handler.NewClosedDaysHandler(closedDaysRepo, log, auditLogger)

	// Initialize services
	cafeteriaService := service.NewCafeteriaService(cafeteriaRepo, log)
	reservationService := service.NewReservationService(
		reservationRepo,
		cafeteriaRepo,
		studentCacheRepo,
		closedDaysRepo,
		paymentClient,
		cfg,
		log,
	)
	menuService := service.NewMenuService(menuRepo, log)

	// Initialize handler
	mealHandler := handler.NewMealHandler(cafeteriaService, reservationService, menuService, log)

	// Initialize background workers
	reservationWorker := worker.NewReservationWorker(reservationRepo, log)
	outboxWorker := worker.NewOutboxWorker(
		outboxRepo,
		publisher,
		cfg.Outbox.PollIntervalSeconds,
		cfg.Outbox.BatchSize,
		cfg.Outbox.MaxRetries,
		log,
	)

	// Start background workers
	reservationWorker.Start(ctx)
	outboxWorker.Start(ctx)

	// Initialize event consumers
	studentConsumer := worker.NewStudentEventConsumer(studentCacheRepo, processedEventsRepo, log)
	paymentConsumer := worker.NewPaymentEventConsumer(reservationRepo, processedEventsRepo, log)

	// Setup RabbitMQ consumers
	setupConsumers(rabbitConn, studentConsumer, paymentConsumer, log)

	// Setup HTTP server
	router := setupRouter(cfg, mealHandler, timeHandler, closedDaysHandler, log)

	// Health: liveness (process up). Ready: deps reachable.
	healthChecks := map[string]sharedHandler.HealthCheck{
		"database": dbPool.Ping,
		"rabbitmq": rabbitConn.Ping,
	}
	if redisClient != nil {
		healthChecks["redis"] = redisClient.Ping
	}
	router.GET("/health", sharedHandler.LivenessHandler("meal-service"))
	router.GET("/ready", sharedHandler.ReadinessHandler("meal-service", healthChecks))

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start HTTP server in goroutine
	go func() {
		log.Info("starting HTTP server", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start HTTP server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	// Stop workers
	reservationWorker.Stop()
	outboxWorker.Stop()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", zap.Error(err))
	}

	log.Info("server exited")
}

func setupRouter(cfg *config.Config, handler *handler.MealHandler, timeHandler *sharedHandler.TimeHandler, closedDaysHandler *handler.ClosedDaysHandler, log *zap.Logger) *gin.Engine {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORS())
	router.Use(middleware.IPRateLimit())
	router.Use(middleware.SetCSRFToken(cfg.Server.Environment == "production"))

	// Health endpoints registered in main() with dependency checks

	// API routes
	api := router.Group("/api/meals")
	{
		// Public routes (no auth required, not protected by Traefik forward-auth)
		api.GET("/menu/monthly", handler.GetMonthlyMenu)

		// Authenticated routes - protected via Traefik forward-auth
		// User info is extracted from X-User-* headers set by Traefik
		auth := api.Group("")
		auth.Use(middleware.ExtractUserFromHeaders())
		auth.Use(middleware.CSRFProtection())
		auth.Use(middleware.UserRateLimit())
		{
			// Cafeterias (all authenticated users can view)
			auth.GET("/cafeterias", handler.GetCafeterias)

			// Admin only routes
			admin := auth.Group("/admin")
			admin.Use(middleware.RequireAdmin())
			{
				admin.POST("/cafeterias", handler.CreateCafeteria)
				admin.PUT("/cafeterias/:cafeteria_id", handler.UpdateCafeteria)
				admin.DELETE("/cafeterias/:cafeteria_id", handler.DeleteCafeteria)
				admin.GET("/cafeterias/:cafeteria_id/qr", handler.GenerateQR)
				admin.POST("/menu/monthly", handler.CreateMonthlyMenu)

				// Time Machine & Closed Days
				timeHandler.RegisterRoutes(admin)
				closedDaysHandler.RegisterRoutes(admin)
			}

			// Student only routes
			student := auth.Group("")
			student.Use(middleware.RequireStudent())
			{
				student.POST("/reservations", handler.CreateReservation)
				student.POST("/reservations/batch", handler.CreateBatchReservation)
				student.GET("/reservations/my", handler.GetMyReservations)
				student.DELETE("/reservations/:reservation_id", handler.CancelReservation)
				student.POST("/reservations/use", handler.UseReservation)
			}
		}
	}

	// Internal routes (service-to-service, no auth)
	internal := router.Group("/api/meals/internal")
	internal.Use(middleware.StripInternalHeaders())
	{
		closedDaysHandler.RegisterInternalRoutes(internal)
	}

	return router
}

func setupConsumers(
	conn *rabbitmq.Connection,
	studentConsumer *worker.StudentEventConsumer,
	paymentConsumer *worker.PaymentEventConsumer,
	log *zap.Logger,
) {
	ch := conn.Channel()

	// Declare required exchanges (auto-declare)
	exchanges := []string{"student.events", "payment.events"}
	for _, exchange := range exchanges {
		if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
			log.Fatal("failed to declare exchange", zap.String("exchange", exchange), zap.Error(err))
		}
	}

	// Student events consumer
	studentEventsConsumer := rabbitmq.NewConsumer(conn)

	// Declare queue
	if err := studentEventsConsumer.DeclareQueue("meal.student.events.queue"); err != nil {
		log.Fatal("failed to declare student events queue", zap.Error(err))
	}

	// Bind routing keys
	studentEventsConsumer.BindQueue("meal.student.events.queue", "student.events", "student.created")
	studentEventsConsumer.BindQueue("meal.student.events.queue", "student.events", "student.updated")
	studentEventsConsumer.BindQueue("meal.student.events.queue", "student.events", "student.deactivated")

	// Consume messages
	studentEventsConsumer.Consume("meal.student.events.queue", func(body []byte) error {
		// We need to determine the routing key from the message content
		// For simplicity, we'll try all handlers
		var event struct {
			EventType string `json:"event_type"`
		}
		if err := rabbitmq.UnmarshalEvent(body, &event); err != nil {
			return err
		}

		switch event.EventType {
		case "student.created":
			return studentConsumer.HandleStudentCreated(context.Background(), body)
		case "student.updated":
			return studentConsumer.HandleStudentUpdated(context.Background(), body)
		case "student.deactivated":
			return studentConsumer.HandleStudentDeactivated(context.Background(), body)
		default:
			log.Warn("unknown event type", zap.String("event_type", event.EventType))
			return nil
		}
	})

	// Payment events consumer
	paymentEventsConsumer := rabbitmq.NewConsumer(conn)

	// Declare queue
	if err := paymentEventsConsumer.DeclareQueue("meal.payment.events.queue"); err != nil {
		log.Fatal("failed to declare payment events queue", zap.Error(err))
	}

	// Bind routing keys
	paymentEventsConsumer.BindQueue("meal.payment.events.queue", "payment.events", "payment.completed")
	paymentEventsConsumer.BindQueue("meal.payment.events.queue", "payment.events", "payment.failed")

	// Consume messages
	paymentEventsConsumer.Consume("meal.payment.events.queue", func(body []byte) error {
		var event struct {
			EventType string `json:"event_type"`
		}
		if err := rabbitmq.UnmarshalEvent(body, &event); err != nil {
			return err
		}

		switch event.EventType {
		case "payment.completed":
			return paymentConsumer.HandlePaymentCompleted(context.Background(), body)
		case "payment.failed":
			return paymentConsumer.HandlePaymentFailed(context.Background(), body)
		default:
			log.Warn("unknown event type", zap.String("event_type", event.EventType))
			return nil
		}
	})

	log.Info("RabbitMQ consumers started")
}
