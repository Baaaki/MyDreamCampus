package main

import (
	grades "github.com/baaaki/mydreamcampus/grades/internal"
	"github.com/baaaki/mydreamcampus/grades/internal/service"
	"github.com/baaaki/mydreamcampus/grades/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/bootstrap"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"go.uber.org/zap"
)

func main() {
	rt := bootstrap.Init(bootstrap.Options{
		Service:       "grades",
		NeedsDatabase: true,
	})

	rt.DeclareQueues([]eventbus.DownstreamBinding{
		// Local student/course/registration projection.
		{Queue: worker.QueueSyncEvents, Exchange: "student.events", RoutingKey: "student.created"},
		{Queue: worker.QueueSyncEvents, Exchange: "student.events", RoutingKey: "student.updated"},
		{Queue: worker.QueueSyncEvents, Exchange: "student.events", RoutingKey: "student.deactivated"},
		{Queue: worker.QueueSyncEvents, Exchange: "course_catalog.events", RoutingKey: "course.semester.created"},
		{Queue: worker.QueueSyncEvents, Exchange: "enrollment.events", RoutingKey: "enrollment.program.approved"},
		{Queue: worker.QueueSyncEvents, Exchange: "attendance.events", RoutingKey: "attendance.semester.failed"},
		// Self-loop: finalizing a whole course runs off the request path.
		{Queue: worker.QueueFinalizeRequested, Exchange: "grades.events", RoutingKey: "grade.finalize.requested"},
		{Queue: worker.QueuePeriodEvents, Exchange: "course_catalog.events",
			RoutingKey: events.PeriodEventRoutingPattern(platformRepo.PeriodTypeGrading)},
	})

	periodRepo := platformRepo.NewSimplePeriodRepository(rt.Pool, "grades")

	module := grades.New(rt.Pool, rt.Rabbit, periodRepo,
		service.NewHTTPSemesterClient(rt.InternalClient("catalog")))
	if err := module.Bootstrap(rt.Ctx); err != nil {
		logger.Fatal("failed to bootstrap grades module", zap.Error(err))
	}

	rt.StartOutbox("grades.events", module.OutboxStore())
	rt.StartRetention(module.RetentionStore())

	logger.Info("grades service ready", zap.String("port", rt.Cfg.Server.Port))
	rt.Run(module)
}
