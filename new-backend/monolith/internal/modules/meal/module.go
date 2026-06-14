package meal

import (
	"context"

	"time"

	"github.com/baaaki/mydreamcampus/monolith/config"
	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/handler"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/worker"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/audit"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Module struct {
	pool        *pgxpool.Pool
	redis       *redis.Client
	cfg         *config.Config
	logger      *zap.Logger
	auditLogger audit.Logger
	rabbitConn  *rabbitmq.Connection

	outboxStore *repository.OutboxStore

	paymentClient service.PaymentClient

	studentConsumer   *worker.StudentEventConsumer
	paymentConsumer   *worker.PaymentEventConsumer
	reservationWorker *worker.ReservationWorker

	mealHandler       *handler.MealHandler
	closedDaysHandler *handler.ClosedDaysHandler
}

func New(
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
	auditLogger audit.Logger,
	rabbitConn *rabbitmq.Connection,
	paymentClient service.PaymentClient,
) *Module {
	return &Module{
		pool:          pool,
		redis:         redisClient,
		cfg:           cfg,
		logger:        logger,
		auditLogger:   auditLogger,
		rabbitConn:    rabbitConn,
		paymentClient: paymentClient,
	}
}

// Name is the URL slug under /api.
func (m *Module) Name() string { return "meals" }

func (m *Module) Bootstrap(ctx context.Context) error {
	// Repositories
	cafeteriaRepo := repository.NewCafeteriaRepository(m.pool)

	closedDaysBaseRepo := repository.NewClosedDaysRepository(m.pool)
	closedDaysRepo := repository.NewClosedDaysCache(closedDaysBaseRepo, time.Hour*24)

	menuRepo := repository.NewMenuRepository(m.pool)
	outboxRepo := repository.NewOutboxRepository(m.pool)
	processedEventsRepo := repository.NewProcessedEventsRepository(m.pool)
	reservationRepo := repository.NewReservationRepository(m.pool)
	studentCacheRepo := repository.NewStudentCacheRepository(m.pool)

	m.outboxStore = repository.NewOutboxStore(outboxRepo)

	// Clients (now injected via New)

	// Services
	cafeteriaSvc := service.NewCafeteriaService(cafeteriaRepo, m.logger)
	menuSvc := service.NewMenuService(menuRepo, m.logger)
	reservationSvc := service.NewReservationService(
		reservationRepo,
		cafeteriaRepo,
		studentCacheRepo,
		closedDaysRepo,
		m.paymentClient,
		m.cfg,
		m.logger,
	)

	// Handlers
	m.mealHandler = handler.NewMealHandler(cafeteriaSvc, reservationSvc, menuSvc, m.logger)
	m.closedDaysHandler = handler.NewClosedDaysHandler(closedDaysRepo, m.logger, m.auditLogger)

	// Workers
	m.studentConsumer = worker.NewStudentEventConsumer(studentCacheRepo, processedEventsRepo, m.logger)
	m.paymentConsumer = worker.NewPaymentEventConsumer(reservationRepo, processedEventsRepo, m.logger)
	m.reservationWorker = worker.NewReservationWorker(reservationRepo, m.logger)

	// Start reservation worker jobs
	m.reservationWorker.Start(ctx)

	consumer := rabbitmq.NewConsumer(m.rabbitConn)

	// Subscribe to student events
	err := consumer.DeclareQueue("meal.student_created_queue")
	if err != nil { return err }
	err = consumer.BindQueue("meal.student_created_queue", "student.events", "student.created")
	if err != nil { return err }
	err = consumer.Consume("meal.student_created_queue", func(body []byte) error { return m.studentConsumer.HandleStudentCreated(ctx, body) })
	if err != nil {
		return err
	}
	err = consumer.DeclareQueue("meal.student_updated_queue")
	if err != nil { return err }
	err = consumer.BindQueue("meal.student_updated_queue", "student.events", "student.updated")
	if err != nil { return err }
	err = consumer.Consume("meal.student_updated_queue", func(body []byte) error { return m.studentConsumer.HandleStudentUpdated(ctx, body) })
	if err != nil {
		return err
	}
	err = consumer.DeclareQueue("meal.student_deactivated_queue")
	if err != nil { return err }
	err = consumer.BindQueue("meal.student_deactivated_queue", "student.events", "student.deactivated")
	if err != nil { return err }
	err = consumer.Consume("meal.student_deactivated_queue", func(body []byte) error { return m.studentConsumer.HandleStudentDeactivated(ctx, body) })
	if err != nil {
		return err
	}

	// Subscribe to payment events
	err = consumer.DeclareQueue("meal.payment_completed_queue")
	if err != nil { return err }
	err = consumer.BindQueue("meal.payment_completed_queue", "payment.events", "payment.completed")
	if err != nil { return err }
	err = consumer.Consume("meal.payment_completed_queue", func(body []byte) error { return m.paymentConsumer.HandlePaymentCompleted(ctx, body) })
	if err != nil {
		return err
	}
	err = consumer.DeclareQueue("meal.payment_failed_queue")
	if err != nil { return err }
	err = consumer.BindQueue("meal.payment_failed_queue", "payment.events", "payment.failed")
	if err != nil { return err }
	err = consumer.Consume("meal.payment_failed_queue", func(body []byte) error { return m.paymentConsumer.HandlePaymentFailed(ctx, body) })
	if err != nil {
		return err
	}

	return nil
}

func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	m.mealHandler.RegisterRoutes(router)
	m.closedDaysHandler.RegisterRoutes(router)
	m.closedDaysHandler.RegisterInternalRoutes(router.Group("/internal"))
}

func (m *Module) OutboxStore() eventbus.OutboxStore {
	return m.outboxStore
}
