package main

import (
	auth "github.com/baaaki/mydreamcampus/auth/internal"
	"github.com/baaaki/mydreamcampus/shared/bootstrap"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"go.uber.org/zap"
)

func main() {
	rt := bootstrap.Init(bootstrap.Options{
		Service:       "auth",
		NeedsDatabase: true,
		// Auth is the only service that cannot run without Redis: the login
		// rate limit and the token blacklist are both fail-closed, so serving
		// without it would mean serving logins with neither check.
		RedisFatal: true,
		// Login, refresh and password are brute-force vectors, and those
		// routes only exist here — no other service needs the buckets.
		LoginRateLimits: true,
	})

	// Auth mirrors staff and student lifecycle into its user table.
	rt.DeclareQueues([]eventbus.DownstreamBinding{
		{Queue: events.QueueAuthStaffEvents, Exchange: "staff.events", RoutingKey: "staff.created"},
		{Queue: events.QueueAuthStaffEvents, Exchange: "staff.events", RoutingKey: "staff.updated"},
		{Queue: events.QueueAuthStaffEvents, Exchange: "staff.events", RoutingKey: "staff.deactivated"},
		{Queue: events.QueueAuthStaffEvents, Exchange: "student.events", RoutingKey: "student.created"},
		{Queue: events.QueueAuthStaffEvents, Exchange: "student.events", RoutingKey: "student.updated"},
		{Queue: events.QueueAuthStaffEvents, Exchange: "student.events", RoutingKey: "student.deactivated"},
	})

	// Before the first login can arrive — see utils.WarmDummyPassword.
	utils.WarmDummyPassword()

	module := auth.New(rt.Cfg, rt.Pool, rt.Redis, rt.Rabbit)
	if err := module.Bootstrap(rt.Ctx); err != nil {
		logger.Fatal("failed to bootstrap auth module", zap.Error(err))
	}

	rt.StartOutbox("auth.events", module.OutboxStore())
	rt.StartRetention(module.RetentionStore())

	logger.Info("auth service ready", zap.String("port", rt.Cfg.Server.Port))
	rt.Run(module)
}
