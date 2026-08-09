package main

import (
	attendance "github.com/baaaki/mydreamcampus/attendance/internal"
	"github.com/baaaki/mydreamcampus/attendance/internal/service"
	"github.com/baaaki/mydreamcampus/attendance/internal/worker"
	"github.com/baaaki/mydreamcampus/shared/bootstrap"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"go.uber.org/zap"
)

func main() {
	rt := bootstrap.Init(bootstrap.Options{
		Service:       "attendance",
		NeedsDatabase: true,
		// Redis is a data store here, not just the rate-limit backend: QR
		// scans buffer in it before they reach the database. Starting without
		// it would mean accepting scans that go nowhere.
		RedisFatal: true,
	})

	rt.DeclareQueues([]eventbus.DownstreamBinding{
		// Local student/course/enrollment projection, so taking attendance
		// never has to call another service on the request path.
		{Queue: worker.QueueSyncEvents, Exchange: "student.events", RoutingKey: "student.created"},
		{Queue: worker.QueueSyncEvents, Exchange: "student.events", RoutingKey: "student.updated"},
		{Queue: worker.QueueSyncEvents, Exchange: "student.events", RoutingKey: "student.deactivated"},
		{Queue: worker.QueueSyncEvents, Exchange: "course_catalog.events", RoutingKey: "course.semester.created"},
		{Queue: worker.QueueSyncEvents, Exchange: "enrollment.events", RoutingKey: "enrollment.program.approved"},
		{Queue: worker.QueuePeriodEvents, Exchange: "course_catalog.events",
			RoutingKey: events.PeriodEventRoutingPattern(platformRepo.PeriodTypeAttendance)},
	})

	periodRepo := platformRepo.NewSimplePeriodRepository(rt.Pool, "attendance")

	module := attendance.New(rt.Cfg, rt.Pool, rt.Redis.Client(), rt.Rabbit,
		service.NewHTTPSemesterClient(rt.InternalClient("catalog")), periodRepo)
	if err := module.Bootstrap(rt.Ctx); err != nil {
		logger.Fatal("failed to bootstrap attendance module", zap.Error(err))
	}

	rt.StartOutbox("attendance.events", module.OutboxStore())
	rt.StartRetention(module.RetentionStore())

	logger.Info("attendance service ready", zap.String("port", rt.Cfg.Server.Port))
	rt.Run(module)
}
