package main

import (
	payment "github.com/baaaki/mydreamcampus/payment/internal"
	"github.com/baaaki/mydreamcampus/shared/bootstrap"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"go.uber.org/zap"
)

func main() {
	// Payment is the one service without a database: it is a mock provider
	// that answers over internal REST and publishes payment.* directly. With
	// no schema there is no outbox table and nothing to retain.
	rt := bootstrap.Init(bootstrap.Options{
		Service:       "payment",
		NeedsDatabase: false,
	})

	module := payment.New(rt.Cfg, logger.Log, rt.Rabbit)
	if err := module.Bootstrap(rt.Ctx); err != nil {
		logger.Fatal("failed to bootstrap payment module", zap.Error(err))
	}

	logger.Info("payment service ready", zap.String("port", rt.Cfg.Server.Port))
	rt.Run(module)
}
