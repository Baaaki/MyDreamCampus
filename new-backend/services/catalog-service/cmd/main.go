package main

import (
	catalog "github.com/baaaki/mydreamcampus/catalog/internal"
	"github.com/baaaki/mydreamcampus/catalog/internal/service"
	"github.com/baaaki/mydreamcampus/catalog/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/bootstrap"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"go.uber.org/zap"
)

func main() {
	rt := bootstrap.Init(bootstrap.Options{
		Service:       "catalog",
		NeedsDatabase: true,
	})

	// Catalog owns audit_log; grades and meal publish their entries here
	// rather than writing across a database boundary that no longer exists.
	rt.DeclareQueues([]eventbus.DownstreamBinding{
		{Queue: worker.QueueAuditEvents, Exchange: "grades.events", RoutingKey: audit.EventAuditEntryCreated},
		{Queue: worker.QueueAuditEvents, Exchange: "meal.events", RoutingKey: audit.EventAuditEntryCreated},
	})

	module := catalog.New(rt.Cfg, rt.Pool, rt.Rabbit,
		service.NewHTTPStaffClient(rt.InternalClient("staff")),
		service.NewHTTPMealClient(rt.InternalClient("meal")),
	)
	if err := module.Bootstrap(rt.Ctx); err != nil {
		logger.Fatal("failed to bootstrap catalog module", zap.Error(err))
	}

	rt.StartOutbox("course_catalog.events", module.OutboxStore())
	rt.StartRetention(module.RetentionStore())

	logger.Info("catalog service ready", zap.String("port", rt.Cfg.Server.Port))
	rt.Run(module)
}
