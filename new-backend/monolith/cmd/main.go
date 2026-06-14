package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/config"
	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	monolithHTTP "github.com/baaaki/mydreamcampus/monolith/internal/http"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth"
	coursecatalog "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog"
	catalogService "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment"
	enrollmentService "github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal"
	mealService "github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/payment"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/student"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/audit"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/database"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/monolith/internal/platform/middleware"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	platformRedis "github.com/baaaki/mydreamcampus/monolith/internal/platform/redis"
	"go.uber.org/zap"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	
	// Initialize JWT secret globally to avoid os.Setenv anti-pattern
	utils.InitJWTSecret(cfg.JWT.Secret)

	if err := logger.Init(cfg.Server.Environment); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	audit.InitSecurity(cfg.Server.Environment)
	defer audit.SyncSecurity()

	logger.Info("starting monolith",
		zap.String("environment", cfg.Server.Environment),
		zap.String("port", cfg.Server.Port),
	)

	pool, err := database.NewPostgresPool(cfg.Database.URL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()
	logger.Info("database connection established")

	redisClient, err := platformRedis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		// Auth depends on Redis for blacklist + rate limiter fail-closed
		// on login/refresh/password. Treat unavailability as fatal.
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()
	logger.Info("Redis connection established")

	if cfg.RateLimit.Enabled {
		rl := platformMiddleware.RateLimitConfig{
			Enabled:     true,
			ServiceName: "monolith",
			IPLimit:     cfg.RateLimit.IPLimit,
			IPWindow:    time.Duration(cfg.RateLimit.IPWindowSecs) * time.Second,
			UserLimit:   cfg.RateLimit.UserLimit,
			UserWindow:  time.Duration(cfg.RateLimit.UserWindowSecs) * time.Second,
			// Auth-specific endpoint limits — login/refresh/password are
			// brute-force vectors so FailClosed is mandatory: when Redis is
			// unreachable we'd rather return 503 than allow unbounded tries.
			EndpointLimits: map[string]platformMiddleware.EndpointLimit{
				"login":    {Limit: cfg.RateLimit.LoginLimit, Window: time.Duration(cfg.RateLimit.LoginWindowSecs) * time.Second, FailClosed: true},
				"refresh":  {Limit: cfg.RateLimit.RefreshLimit, Window: time.Duration(cfg.RateLimit.RefreshWindowSecs) * time.Second, FailClosed: true},
				"password": {Limit: cfg.RateLimit.PasswordLimit, Window: time.Duration(cfg.RateLimit.PasswordWindowSecs) * time.Second, FailClosed: true},
			},
		}
		platformMiddleware.SetRateLimiter(platformMiddleware.NewRateLimiter(redisClient, rl))
		logger.Info("rate limiter configured",
			zap.Int("ip_limit", cfg.RateLimit.IPLimit),
			zap.Int("user_limit", cfg.RateLimit.UserLimit),
		)
	}

	rabbitConn, err := rabbitmq.NewConnection(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Fatal("failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rabbitConn.Close()
	logger.Info("RabbitMQ connection established")

	publisher := rabbitmq.NewPublisher(rabbitConn)
	if err := eventbus.DeclareModuleExchanges(publisher); err != nil {
		logger.Fatal("failed to declare module exchanges", zap.Error(err))
	}
	logger.Info("module exchanges declared", zap.Int("count", len(eventbus.ModuleExchanges)))

	// Downstream queue bindings — pre-declared so messages persist even
	// when consumers are offline (plan section 5.6.3). Each module appends
	// its consumers as it migrates. Auth + student still consume staff
	// events from RabbitMQ until those modules switch to in-process pubsub.
	downstreamBindings := []eventbus.DownstreamBinding{
		// auth — keeps user records in sync with staff/student lifecycle.
		{Queue: "auth_events_queue", Exchange: "staff.events", RoutingKey: "staff.created"},
		{Queue: "auth_events_queue", Exchange: "staff.events", RoutingKey: "staff.updated"},
		{Queue: "auth_events_queue", Exchange: "staff.events", RoutingKey: "staff.deactivated"},
		{Queue: "auth_events_queue", Exchange: "student.events", RoutingKey: "student.created"},
		{Queue: "auth_events_queue", Exchange: "student.events", RoutingKey: "student.updated"},
		{Queue: "auth_events_queue", Exchange: "student.events", RoutingKey: "student.deactivated"},
		// student — drops advisor assignment when the staff member is removed.
		{Queue: "student.staff_events", Exchange: "staff.events", RoutingKey: "staff.deactivated"},
	}
	if err := eventbus.DeclareDownstreamBindings(publisher, downstreamBindings); err != nil {
		logger.Fatal("failed to declare downstream bindings", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Per-module outbox workers (plan section 5.5.2 selection A) start here
	// as each module migrates. Auth does not yet publish events to its
	// outbox — that wiring lands with notification (Faz 3).
	//
	//   go eventbus.NewOutboxWorker(
	//       "auth", "auth.events", authModule.OutboxStore(), publisher,
	//       cfg.OutboxInterval(), int32(cfg.Outbox.BatchSize),
	//   ).Start(ctx)

	authModule := auth.New(cfg, pool, redisClient, rabbitConn)
	if err := authModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap auth module", zap.Error(err))
	}

	staffModule := staff.New(pool)
	studentModule := student.New(pool, rabbitConn, staffModule.StaffService())
	if err := studentModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap student module", zap.Error(err))
	}
	catalogModule := coursecatalog.New(cfg, pool, staffModule.StaffService())

	enrollmentStudentClient := enrollmentService.NewInProcessStudentClient(studentModule.StudentService())
	enrollmentCourseClient := enrollmentService.NewInProcessCourseCatalogClient(catalogModule.SemesterService())

	enrollmentModule := enrollment.New(pool, enrollmentStudentClient, enrollmentCourseClient, catalogModule.PeriodRepo())
	if err := enrollmentModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap enrollment module", zap.Error(err))
	}

	attendanceModule := attendance.New(cfg, pool, redisClient.Client(), catalogModule.SemesterService(), catalogModule.PeriodRepo())

	gradesAuditLogger := catalogService.NewDirectAuditLogger(catalogModule.AuditRepo(), "grades")
	gradesModule := grades.New(pool, catalogModule.PeriodRepo(), gradesAuditLogger, catalogModule.SemesterService())
	if err := gradesModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap grades module", zap.Error(err))
	}

	paymentModule := payment.New(logger.Log, rabbitConn)
	if err := paymentModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap payment module", zap.Error(err))
	}

	mealAuditLogger := catalogService.NewDirectAuditLogger(catalogModule.AuditRepo(), "meal")
	paymentAdapter := mealService.NewPaymentAdapter(paymentModule.PaymentService())
	mealModule := meal.New(pool, redisClient.Client(), cfg, logger.Log, mealAuditLogger, rabbitConn, paymentAdapter)
	if err := mealModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap meal module", zap.Error(err))
	}

	// Per-module outbox workers. Auth does not yet publish events
	// (notification arrives in Faz 3); wait, auth now publishes events!
	outboxInterval := time.Duration(cfg.Outbox.IntervalSeconds) * time.Second
	batchSize := int32(cfg.Outbox.BatchSize)
	go eventbus.NewOutboxWorker("auth", "auth.events", authModule.OutboxStore(),
		publisher, outboxInterval, batchSize).Start(ctx)
	go eventbus.NewOutboxWorker("staff", "staff.events", staffModule.OutboxStore(),
		publisher, outboxInterval, batchSize).Start(ctx)
	go eventbus.NewOutboxWorker("student", "student.events", studentModule.OutboxStore(),
		publisher, outboxInterval, batchSize).Start(ctx)
	go eventbus.NewOutboxWorker("course_catalog", "course_catalog.events", catalogModule.OutboxStore(),
		publisher, outboxInterval, batchSize).Start(ctx)
	go eventbus.NewOutboxWorker("enrollment", "enrollment.events", enrollmentModule.OutboxStore(),
		publisher, outboxInterval, batchSize).Start(ctx)
	go eventbus.NewOutboxWorker("attendance", "attendance.events", attendanceModule.OutboxStore(),
		publisher, outboxInterval, batchSize).Start(ctx)
	go eventbus.NewOutboxWorker("grades", "grades.events", gradesModule.OutboxStore(),
		publisher, outboxInterval, batchSize).Start(ctx)
	go eventbus.NewOutboxWorker("meal", "meal.events", mealModule.OutboxStore(),
		publisher, outboxInterval, batchSize).Start(ctx)

	server := monolithHTTP.NewServer(cfg)
	server.RegisterHealthCheck("database", pool.Ping)
	server.RegisterHealthCheck("rabbitmq", rabbitConn.Ping)
	server.RegisterHealthCheck("redis", redisClient.Ping)

	server.RegisterModules(authModule, staffModule, studentModule, catalogModule, enrollmentModule, attendanceModule, gradesModule, paymentModule, mealModule)
	server.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down monolith")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}
	logger.Info("monolith exited")
}
