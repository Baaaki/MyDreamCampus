package main

import (
	meal "github.com/baaaki/mydreamcampus/meal/internal"
	"github.com/baaaki/mydreamcampus/meal/internal/service"
	"github.com/baaaki/mydreamcampus/shared/bootstrap"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"go.uber.org/zap"
)

func main() {
	rt := bootstrap.Init(bootstrap.Options{
		Service:       "meal",
		NeedsDatabase: true,
		// Redis is a data store here, not just the rate-limit backend: the
		// cafeteria QR flow reads and writes it directly.
		RedisFatal: true,
		// Meal QR codes are HMAC'd with QR_SECRET.
		SignsQRCodes: true,
	})

	// Meal declares its own queues. Payment used to declare the two
	// payment_* ones — a publisher defining its consumer's topology, which
	// kept them alive even with no meal service running.
	rt.DeclareQueues([]eventbus.DownstreamBinding{
		{Queue: "meal.student_created_queue", Exchange: "student.events", RoutingKey: "student.created"},
		{Queue: "meal.student_updated_queue", Exchange: "student.events", RoutingKey: "student.updated"},
		{Queue: "meal.student_deactivated_queue", Exchange: "student.events", RoutingKey: "student.deactivated"},
		{Queue: "meal.payment_completed_queue", Exchange: "payment.events", RoutingKey: "payment.completed"},
		{Queue: "meal.payment_failed_queue", Exchange: "payment.events", RoutingKey: "payment.failed"},
	})

	module := meal.New(rt.Pool, rt.Redis.Client(), rt.Cfg, logger.Log, rt.Rabbit,
		service.NewHTTPPaymentClient(rt.InternalClient("payment")))
	if err := module.Bootstrap(rt.Ctx); err != nil {
		logger.Fatal("failed to bootstrap meal module", zap.Error(err))
	}

	rt.StartOutbox("meal.events", module.OutboxStore())
	rt.StartRetention(module.RetentionStore())

	logger.Info("meal service ready", zap.String("port", rt.Cfg.Server.Port))
	rt.Run(module)
}
