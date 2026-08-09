package main

import (
	"github.com/baaaki/mydreamcampus/shared/bootstrap"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	student "github.com/baaaki/mydreamcampus/student/internal"
	"github.com/baaaki/mydreamcampus/student/internal/service"
	"go.uber.org/zap"
)

func main() {
	rt := bootstrap.Init(bootstrap.Options{
		Service:       "student",
		NeedsDatabase: true,
	})

	// Drops the advisor assignment when a staff member is deactivated.
	rt.DeclareQueues([]eventbus.DownstreamBinding{
		{Queue: events.QueueStudentStaffEvents, Exchange: "staff.events", RoutingKey: "staff.deactivated"},
	})

	staffClient := service.NewHTTPStaffClient(rt.InternalClient("staff"))
	module := student.New(rt.Cfg, rt.Pool, rt.Rabbit, staffClient)
	if err := module.Bootstrap(rt.Ctx); err != nil {
		logger.Fatal("failed to bootstrap student module", zap.Error(err))
	}

	rt.StartOutbox("student.events", module.OutboxStore())
	rt.StartRetention(module.RetentionStore())

	logger.Info("student service ready", zap.String("port", rt.Cfg.Server.Port))
	rt.Run(module)
}
