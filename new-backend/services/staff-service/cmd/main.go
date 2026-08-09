package main

import (
	"github.com/baaaki/mydreamcampus/shared/bootstrap"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	staff "github.com/baaaki/mydreamcampus/staff/internal"
	"go.uber.org/zap"
)

func main() {
	rt := bootstrap.Init(bootstrap.Options{
		Service:       "staff",
		NeedsDatabase: true,
	})

	// Staff consumes nothing: it is the source of staff.* and answers
	// instructor lookups over internal REST. No queues to declare.
	module := staff.New(rt.Cfg, rt.Pool)

	rt.StartOutbox("staff.events", module.OutboxStore())
	rt.StartRetention(module.RetentionStore())

	logger.Info("staff service ready", zap.String("port", rt.Cfg.Server.Port))
	rt.Run(module)
}
