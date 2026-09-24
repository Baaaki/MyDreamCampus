package payment

import (
	"context"
	"time"

	"github.com/baaaki/mydreamcampus/payment/internal/handler"
	"github.com/baaaki/mydreamcampus/payment/internal/repository"
	"github.com/baaaki/mydreamcampus/payment/internal/service"
	"github.com/baaaki/mydreamcampus/payment/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Module struct {
	cfg            *config.Config
	logger         *zap.Logger
	pool           *pgxpool.Pool
	paymentService *service.PaymentService
	paymentHandler *handler.PaymentHandler
}

// expiryInterval matches meal's reservation expiry job, so a lapsed payment
// and its reservation close within the same minute.
const expiryInterval = time.Minute

func New(cfg *config.Config, logger *zap.Logger, pool *pgxpool.Pool) *Module {
	paymentSvc := service.NewPaymentService(
		repository.NewPaymentRepository(pool),
		time.Duration(cfg.Reservation.TimeoutMinutes)*time.Minute,
		logger,
	)

	return &Module{
		cfg:            cfg,
		logger:         logger,
		pool:           pool,
		paymentService: paymentSvc,
		paymentHandler: handler.NewPaymentHandler(paymentSvc),
	}
}

// Name is the URL slug under /api.
func (m *Module) Name() string {
	return "payments"
}

// Bootstrap starts the expiry worker; it stops when ctx is cancelled.
func (m *Module) Bootstrap(ctx context.Context) error {
	go worker.NewExpiryWorker(m.paymentService, expiryInterval, m.logger).Start(ctx)
	return nil
}

// OutboxStore feeds the shared relay that publishes payment.events.
func (m *Module) OutboxStore() eventbus.OutboxStore {
	return repository.NewOutboxStore(repository.NewOutboxRepository(m.pool))
}

// RetentionStore lets the shared retention worker prune relayed events.
func (m *Module) RetentionStore() eventbus.RetentionStore {
	return repository.NewRetentionStore(m.pool)
}

// RegisterRoutes mounts nothing under /api — payment has no user-facing
// endpoints. Meal, its single caller, reaches it through RegisterPublicRoutes.
func (m *Module) RegisterRoutes(*gin.RouterGroup) {}

// RegisterPublicRoutes mounts the /internal sub-tree at the root, outside the
// /api prefix Caddy proxies, so payment is reachable only from inside the
// compose network. The secret check stays as the second line of defence.
func (m *Module) RegisterPublicRoutes(r *gin.Engine) {
	internal := r.Group("/internal")
	internal.Use(platformMiddleware.RequireInternalSecret(m.cfg.Server.InternalSecret))
	m.paymentHandler.RegisterInternalRoutes(internal)
}
