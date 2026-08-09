package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/httpserver"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance"
	attendanceService "github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/service"
	attendanceWorker "github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/worker"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth"
	coursecatalog "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog"
	catalogService "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/service"
	catalogWorker "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/worker"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment"
	enrollmentService "github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/service"
	enrollmentWorker "github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/worker"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades"
	gradesService "github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/service"
	gradesWorker "github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/worker"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal"
	mealService "github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/payment"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/student"
	studentService "github.com/baaaki/mydreamcampus/monolith/internal/modules/student/service"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/database"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/rabbitmq"
	platformRedis "github.com/baaaki/mydreamcampus/shared/platform/redis"
	platformRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"go.uber.org/zap"
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
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Warn("redis close failed", zap.Error(err))
		}
	}()
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
	defer func() {
		if err := rabbitConn.Close(); err != nil {
			logger.Warn("rabbitmq close failed", zap.Error(err))
		}
	}()
	logger.Info("RabbitMQ connection established")

	publisher := rabbitmq.NewPublisher(rabbitConn)
	if err := eventbus.DeclareModuleExchanges(publisher); err != nil {
		logger.Fatal("failed to declare module exchanges", zap.Error(err))
	}
	logger.Info("module exchanges declared", zap.Int("count", len(eventbus.ModuleExchanges)))

	// Downstream queue bindings — pre-declared so messages persist even
	// when consumers are offline. Each module appends
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
		// attendance — local student/course/enrollment cache sync.
		{Queue: "attendance.sync_events", Exchange: "student.events", RoutingKey: "student.created"},
		{Queue: "attendance.sync_events", Exchange: "student.events", RoutingKey: "student.updated"},
		{Queue: "attendance.sync_events", Exchange: "student.events", RoutingKey: "student.deactivated"},
		{Queue: "attendance.sync_events", Exchange: "course_catalog.events", RoutingKey: "course.semester.created"},
		{Queue: "attendance.sync_events", Exchange: "enrollment.events", RoutingKey: "enrollment.program.approved"},
		// grades — cache sync, registrations and attendance failures.
		{Queue: "grades.sync_events", Exchange: "student.events", RoutingKey: "student.created"},
		{Queue: "grades.sync_events", Exchange: "student.events", RoutingKey: "student.updated"},
		{Queue: "grades.sync_events", Exchange: "student.events", RoutingKey: "student.deactivated"},
		{Queue: "grades.sync_events", Exchange: "course_catalog.events", RoutingKey: "course.semester.created"},
		{Queue: "grades.sync_events", Exchange: "enrollment.events", RoutingKey: "enrollment.program.approved"},
		{Queue: "grades.sync_events", Exchange: "attendance.events", RoutingKey: "attendance.semester.failed"},
		// grades — self-loop so AutoFinalize runs off the request path.
		{Queue: "grades.finalize_requested", Exchange: "grades.events", RoutingKey: "grade.finalize.requested"},
		// enrollment — passed-prerequisite projection for enrollment validation.
		{Queue: "enrollment.sync_events", Exchange: "grades.events", RoutingKey: "grade.student.prerequisite.passed"},
		// catalog owns every service's academic period and publishes one event
		// per consumer; each consumer binds only its own period type, so the
		// filtering happens at the broker rather than in the consumer.
		// catalog owns audit_log; grades and meal publish their entries
		// instead of writing across the schema boundary.
		{Queue: catalogWorker.QueueAuditEvents, Exchange: "grades.events", RoutingKey: audit.EventAuditEntryCreated},
		{Queue: catalogWorker.QueueAuditEvents, Exchange: "meal.events", RoutingKey: audit.EventAuditEntryCreated},
		{Queue: enrollmentWorker.QueuePeriodEvents, Exchange: "course_catalog.events", RoutingKey: events.PeriodEventRoutingPattern(platformRepo.PeriodTypeEnrollment)},
		{Queue: gradesWorker.QueuePeriodEvents, Exchange: "course_catalog.events", RoutingKey: events.PeriodEventRoutingPattern(platformRepo.PeriodTypeGrading)},
		{Queue: attendanceWorker.QueuePeriodEvents, Exchange: "course_catalog.events", RoutingKey: events.PeriodEventRoutingPattern(platformRepo.PeriodTypeAttendance)},
	}
	if err := eventbus.DeclareDownstreamBindings(publisher, downstreamBindings); err != nil {
		logger.Fatal("failed to declare downstream bindings", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	authModule := auth.New(cfg, pool, redisClient, rabbitConn)
	if err := authModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap auth module", zap.Error(err))
	}

	// One transport per target service. The closed-day fan-out to meal has
	// always gone over HTTP, so its client is built in both modes.
	transports := newInternalTransports(cfg)
	httpMode := cfg.InternalClient.Mode == config.InternalClientModeHTTP
	logger.Info("internal client mode", zap.String("mode", cfg.InternalClient.Mode))

	staffModule := staff.New(cfg, pool)

	// The cross-module clients are chosen here, once. Everything downstream
	// sees an interface and cannot tell which implementation it got.
	var (
		catalogStaffClient       catalogService.StaffClient
		studentStaffClient       studentService.StaffServiceInterface
		enrollmentStudentClient  enrollmentService.StudentClient
		enrollmentCourseClient   enrollmentService.CourseCatalogClient
		attendanceSemesterClient attendanceService.SemesterClient
		gradesSemesterClient     gradesService.SemesterClient
		mealPaymentClient        mealService.PaymentClient
	)
	if httpMode {
		catalogStaffClient = catalogService.NewHTTPStaffClient(transports.staff)
		studentStaffClient = studentService.NewHTTPStaffClient(transports.staff)
		enrollmentStudentClient = enrollmentService.NewHTTPStudentClient(transports.student)
		enrollmentCourseClient = enrollmentService.NewHTTPCourseCatalogClient(transports.catalog)
		attendanceSemesterClient = attendanceService.NewHTTPSemesterClient(transports.catalog)
		gradesSemesterClient = gradesService.NewHTTPSemesterClient(transports.catalog)
		mealPaymentClient = mealService.NewHTTPPaymentClient(transports.payment)
	}

	studentModule := student.New(cfg, pool, rabbitConn, orInProcess(studentStaffClient,
		func() studentService.StaffServiceInterface {
			return studentService.NewInProcessStaffClient(staffModule.StaffService())
		}))
	if err := studentModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap student module", zap.Error(err))
	}

	catalogModule := coursecatalog.New(cfg, pool, rabbitConn,
		orInProcess(catalogStaffClient, func() catalogService.StaffClient {
			return catalogService.NewInProcessStaffClient(staffModule.StaffService())
		}),
		catalogService.NewHTTPMealClient(transports.meal),
	)
	if err := catalogModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap catalog module", zap.Error(err))
	}

	// Each module reads academic_periods from its own schema. Catalog stays the
	// source of truth and pushes changes as events; nobody reads across a
	// schema boundary, which is what makes the split into separate databases
	// possible.
	enrollmentPeriodRepo := platformRepo.NewSimplePeriodRepository(pool, "enrollment")
	attendancePeriodRepo := platformRepo.NewSimplePeriodRepository(pool, "attendance")
	gradesPeriodRepo := platformRepo.NewSimplePeriodRepository(pool, "grades")

	enrollmentModule := enrollment.New(pool, rabbitConn,
		orInProcess(enrollmentStudentClient, func() enrollmentService.StudentClient {
			return enrollmentService.NewInProcessStudentClient(studentModule.StudentService())
		}),
		orInProcess(enrollmentCourseClient, func() enrollmentService.CourseCatalogClient {
			return enrollmentService.NewInProcessCourseCatalogClient(catalogModule.SemesterService())
		}),
		enrollmentPeriodRepo)
	if err := enrollmentModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap enrollment module", zap.Error(err))
	}

	attendanceModule := attendance.New(cfg, pool, redisClient.Client(), rabbitConn,
		orInProcess(attendanceSemesterClient, func() attendanceService.SemesterClient {
			return attendanceService.NewInProcessSemesterClient(catalogModule.SemesterService())
		}),
		attendancePeriodRepo)
	if err := attendanceModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap attendance module", zap.Error(err))
	}

	gradesModule := grades.New(pool, rabbitConn, gradesPeriodRepo,
		orInProcess(gradesSemesterClient, func() gradesService.SemesterClient {
			return gradesService.NewInProcessSemesterClient(catalogModule.SemesterService())
		}))
	if err := gradesModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap grades module", zap.Error(err))
	}

	paymentModule := payment.New(cfg, logger.Log, rabbitConn)
	if err := paymentModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap payment module", zap.Error(err))
	}

	mealModule := meal.New(pool, redisClient.Client(), cfg, logger.Log, rabbitConn,
		orInProcess(mealPaymentClient, func() mealService.PaymentClient {
			return mealService.NewPaymentAdapter(paymentModule.PaymentService())
		}))
	if err := mealModule.Bootstrap(ctx); err != nil {
		logger.Fatal("failed to bootstrap meal module", zap.Error(err))
	}

	// One outbox worker per publishing module — events are written to the
	// module's outbox table inside the business transaction, then relayed here.
	outboxInterval := time.Duration(cfg.Outbox.IntervalSeconds) * time.Second
	batchSize := utils.ClampToInt32(cfg.Outbox.BatchSize)
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

	server := httpserver.NewServer(cfg)
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
