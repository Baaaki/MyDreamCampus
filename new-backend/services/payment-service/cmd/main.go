package main

import (
	payment "github.com/baaaki/mydreamcampus/payment/internal"
	"github.com/baaaki/mydreamcampus/shared/bootstrap"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"go.uber.org/zap"
)

func main() {
	rt := bootstrap.Init(bootstrap.Options{
		Service:       "payment",
		NeedsDatabase: true,
	})

	// Payment consumes nothing: meal calls it over internal REST and learns
	// the outcome from payment.* events. No queues to declare.
	module := payment.New(rt.Cfg, logger.Log, rt.Pool, rt.Rabbit)
	if err := module.Bootstrap(rt.Ctx); err != nil {
		logger.Fatal("failed to bootstrap payment module", zap.Error(err))
	}

	rt.StartOutbox("payment.events", module.OutboxStore())
	rt.StartRetention(module.RetentionStore())

	logger.Info("payment service ready", zap.String("port", rt.Cfg.Server.Port))
	rt.Run(module)
}
