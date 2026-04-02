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
	"github.com/baaaki/mydreamcampus/shared/logger"
	sharedMiddleware "github.com/baaaki/mydreamcampus/shared/middleware"
	"github.com/baaaki/mydreamcampus/shared/rabbitmq"
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

	// Initialize services
	staffService := service.NewStaffService(staffRepo)

	// Initialize handlers
	staffHandler := handler.NewStaffHandler(staffService)

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
	router := setupRouter(staffHandler, cfg.Server.Environment)

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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown",
			zap.Error(err),
		)
	}

	logger.Info("server exited")
}

func setupRouter(staffHandler *handler.StaffHandler, env string) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(sharedMiddleware.Recovery())
	router.Use(sharedMiddleware.CORS())
	router.Use(sharedMiddleware.RequestLogger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "staff-service",
		})
	})

	// API routes
	api := router.Group("/api/staff")
	{
		// Public routes (no auth required for now)
		api.POST("", staffHandler.CreateStaff)
		api.GET("", staffHandler.ListStaff)
		api.GET("/:id", staffHandler.GetStaffByID)
		api.PUT("/:id", staffHandler.UpdateStaff)
		api.DELETE("/:id", staffHandler.DeleteStaff)
	}

	// Protected routes (with JWT auth)
	// protected := api.Group("")
	// protected.Use(sharedMiddleware.JWTAuth())
	// {
	// 	protected.PUT("/:id", staffHandler.UpdateStaff)
	// 	protected.DELETE("/:id", staffHandler.DeleteStaff)
	// }

	return router
}

func setupRabbitMQ(conn *rabbitmq.Connection) error {
	channel := conn.Channel()

	// Declare staff exchange
	if err := channel.ExchangeDeclare(
		"staff_exchange", // name
		"topic",          // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	logger.Info("RabbitMQ exchange declared",
		zap.String("exchange", "staff_exchange"),
	)

	return nil
}
