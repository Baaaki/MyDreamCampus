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

	eventConsumer     *worker.EventConsumer
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
	studentConsumer := worker.NewStudentEventConsumer(studentCacheRepo, processedEventsRepo, m.logger)
	paymentConsumer := worker.NewPaymentEventConsumer(reservationRepo, processedEventsRepo, m.logger)
	m.reservationWorker = worker.NewReservationWorker(reservationRepo, m.logger)
	
	consumer := rabbitmq.NewConsumer(m.rabbitConn)
	m.eventConsumer = worker.NewEventConsumer(consumer, studentConsumer, paymentConsumer)

	// Start reservation worker jobs
	m.reservationWorker.Start(ctx)

	// Start event consumer
	if err := m.eventConsumer.Start(ctx); err != nil {
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
