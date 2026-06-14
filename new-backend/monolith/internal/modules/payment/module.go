package payment

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/payment/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Module struct {
	logger         *zap.Logger
	paymentService *service.PaymentService
}

func New(logger *zap.Logger, rabbitConn *rabbitmq.Connection) *Module {
	publisher := rabbitmq.NewPublisher(rabbitConn)
	paymentSvc := service.NewPaymentService(publisher, logger)

	return &Module{
		logger:         logger,
		paymentService: paymentSvc,
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

func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	// The mock payment module doesn't expose HTTP routes in this iteration.
}

func (m *Module) PaymentService() *service.PaymentService {
	return m.paymentService
}
