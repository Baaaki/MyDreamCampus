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
	router := setupRouter(studentHandler, cfg.Server.Environment)

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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown",
			zap.Error(err),
		)
	}

	logger.Info("server exited")
}

func setupRouter(studentHandler *handler.StudentHandler, env string) *gin.Engine {
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
			"service": "student-service",
		})
	})

	// API routes with versioning
	v1 := router.Group("/api/v1/students")
	{
		// Student CRUD
		v1.POST("", studentHandler.CreateStudent)
		v1.GET("", studentHandler.ListStudents)
		v1.GET("/:id", studentHandler.GetStudentByID)
		v1.PUT("/:id", studentHandler.UpdateStudent)
		v1.DELETE("/:id", studentHandler.DeleteStudent)

		// Advanced search
		v1.POST("/search", studentHandler.SearchStudents)

		// Advisor-related endpoints
		v1.GET("/my-advisees", studentHandler.GetMyAdvisees)
		v1.GET("/orphaned", studentHandler.ListOrphanedStudents)
		v1.PUT("/bulk-advisor-assign", studentHandler.BulkAssignAdvisor)

		// Bulk import
		v1.POST("/bulk-import", studentHandler.BulkImport)
		v1.GET("/bulk-import/:job_id", studentHandler.GetImportJobStatus)
		v1.GET("/bulk-import", studentHandler.ListImportJobs)
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

	return nil
}
