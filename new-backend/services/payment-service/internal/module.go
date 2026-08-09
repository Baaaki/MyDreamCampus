package payment

import (
	"context"

	"github.com/baaaki/mydreamcampus/payment/internal/handler"
	"github.com/baaaki/mydreamcampus/payment/internal/service"
	"github.com/baaaki/mydreamcampus/shared/config"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/rabbitmq"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Module struct {
	cfg            *config.Config
	logger         *zap.Logger
	paymentService *service.PaymentService
	paymentHandler *handler.PaymentHandler
}

func New(cfg *config.Config, logger *zap.Logger, rabbitConn *rabbitmq.Connection) *Module {
	publisher := rabbitmq.NewPublisher(rabbitConn)
	paymentSvc := service.NewPaymentService(publisher, logger)

	return &Module{
		cfg:            cfg,
		logger:         logger,
		paymentService: paymentSvc,
		paymentHandler: handler.NewPaymentHandler(paymentSvc),
	}
}

// Name is the URL slug under /api.
func (m *Module) Name() string {
	return "payments"
}

func (m *Module) Bootstrap(ctx context.Context) error {
	m.logger.Info("bootstrapping mock payment module")
	return nil
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
