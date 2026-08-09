package payment

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/payment/handler"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/payment/service"
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

// RegisterRoutes mounts only the /internal sub-tree — payment has no
// user-facing endpoints; meal is its single caller.
func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	internal := router.Group("/internal")
	internal.Use(platformMiddleware.RequireInternalSecret(m.cfg.Server.InternalSecret))
	m.paymentHandler.RegisterInternalRoutes(internal)
}

func (m *Module) PaymentService() *service.PaymentService {
	return m.paymentService
}
