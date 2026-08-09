package main

import (
	enrollment "github.com/baaaki/mydreamcampus/enrollment/internal"
	"github.com/baaaki/mydreamcampus/enrollment/internal/service"
	"github.com/baaaki/mydreamcampus/enrollment/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/bootstrap"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"go.uber.org/zap"
)

func main() {
	rt := bootstrap.Init(bootstrap.Options{
		Service:       "enrollment",
		NeedsDatabase: true,
	})

	rt.DeclareQueues([]eventbus.DownstreamBinding{
		// Passed-prerequisite projection, so enrollment validation does not
		// have to ask grades on the request path.
		{Queue: worker.QueueSyncEvents, Exchange: "grades.events", RoutingKey: "grade.student.prerequisite.passed"},
		// Catalog owns every academic period and publishes one event per
		// consumer; the broker filters, so only enrollment periods land here.
		{Queue: worker.QueuePeriodEvents, Exchange: "course_catalog.events",
			RoutingKey: events.PeriodEventRoutingPattern(platformRepo.PeriodTypeEnrollment)},
	})

	// Reads its own academic_periods projection, never catalog's table.
	periodRepo := platformRepo.NewSimplePeriodRepository(rt.Pool, "enrollment")

	module := enrollment.New(rt.Pool, rt.Rabbit,
		service.NewHTTPStudentClient(rt.InternalClient("student")),
		service.NewHTTPCourseCatalogClient(rt.InternalClient("catalog")),
		periodRepo,
	)
	if err := module.Bootstrap(rt.Ctx); err != nil {
		logger.Fatal("failed to bootstrap enrollment module", zap.Error(err))
	}

	rt.StartOutbox("enrollment.events", module.OutboxStore())
	rt.StartRetention(module.RetentionStore())

	logger.Info("enrollment service ready", zap.String("port", rt.Cfg.Server.Port))
	rt.Run(module)
}
