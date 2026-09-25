// Package bootstrap holds the startup sequence every service shares.
//
// It exists because that sequence is not boilerplate: the order matters
// (JWT secret before anything validates a token, logger before the first
// Fatal), and three of its steps are security controls that a per-service
// copy would eventually drop — the shared rate-limit bucket, the audit
// logger, and the refusal to start without an internal secret. Written once,
// they hold for all ten binaries.
package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baaaki/mydreamcampus/shared/client"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/httpserver"
	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/clocksync"
	"github.com/baaaki/mydreamcampus/shared/platform/database"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/rabbitmq"
	platformRedis "github.com/baaaki/mydreamcampus/shared/platform/redis"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Options describes what a service needs from the shared startup.
type Options struct {
	// Service is the log field and the outbox worker's name, e.g. "grades".
	Service string
	// NeedsDatabase opens a Postgres pool from DB_URL.
	NeedsDatabase bool
	// RedisFatal makes an unreachable Redis stop the service. Auth sets it:
	// its login rate limit and token blacklist are fail-closed, so running
	// without Redis would mean serving logins with neither. Everywhere else
	// Redis backs fail-open rate limiting and is not worth a boot failure.
	RedisFatal bool
	// LoginRateLimits adds the per-endpoint buckets for login, refresh and
	// password. Only auth serves those routes; the values come from config.
	LoginRateLimits bool
	// SignsQRCodes refuses to start in production without a real QR_SECRET.
	// Only meal signs with the shared key; attendance uses per-session keys.
	SignsQRCodes bool
}

// Runtime is the wired infrastructure handed back to a service's main.
// Fields a service did not ask for are nil.
type Runtime struct {
	Cfg       *config.Config
	Pool      *pgxpool.Pool
	Redis     *platformRedis.ClientWrapper
	Rabbit    *rabbitmq.Connection
	Publisher *rabbitmq.Publisher
	Server    *httpserver.Server

	// Ctx is cancelled at the start of shutdown so workers and consumers
	// stop before the HTTP server drains.
	Ctx context.Context

	service string
	cancel  context.CancelFunc
	closers []func()
}

// sharedRateLimitBucket is the ServiceName every service passes to the rate
// limiter. It is deliberately not the service's own name: the key is
// "ratelimit:<name>:ip:<ip>", so nine names would mean nine buckets and an
// attacker could spread requests across services to multiply the effective
// limit by nine.
const sharedRateLimitBucket = "public"

// Init runs the startup sequence and exits the process on any failure.
// It is only ever called from main, where a Fatal is the correct response.
func Init(opts Options) *Runtime {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Before anything can validate a token.
	utils.InitJWTSecret(cfg.JWT.Secret)

	if err := logger.Init(cfg.Server.Environment); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	// Ten services write into one log stream; without this field there is no
	// way to tell whose line it is. Retrofitting it means touching ten mains.
	logger.Log = logger.Log.With(zap.String("service", opts.Service))

	audit.InitSecurity(cfg.Server.Environment)

	if opts.SignsQRCodes {
		if err := cfg.ValidateQRSecret(); err != nil {
			logger.Fatal("invalid QR configuration", zap.Error(err))
		}
	}

	if cfg.Server.InternalSecret == "" {
		// An empty secret makes RequireInternalSecret a no-op, which would
		// leave every /internal route open to anything on the network.
		logger.Fatal("INTERNAL_SERVICE_SECRET is required")
	}

	logger.Info("starting service",
		zap.String("environment", cfg.Server.Environment),
		zap.String("port", cfg.Server.Port),
	)

	ctx, cancel := context.WithCancel(context.Background())
	rt := &Runtime{Cfg: cfg, Ctx: ctx, service: opts.Service, cancel: cancel}
	rt.defer_(func() { logger.Sync() })
	rt.defer_(audit.SyncSecurity)

	if opts.NeedsDatabase {
		// Checked here rather than in config.Validate, so only the services
		// that open a pool require it.
		if cfg.Database.URL == "" {
			logger.Fatal("DB_URL is required")
		}
		pool, err := database.NewPostgresPool(cfg.Database.URL, cfg.Database.MaxConns)
		if err != nil {
			logger.Fatal("failed to connect to database", zap.Error(err))
		}
		rt.Pool = pool
		rt.defer_(pool.Close)
		logger.Info("database connection established")
	}

	rt.initRedis(opts)

	rabbitConn, err := rabbitmq.NewConnection(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Fatal("failed to connect to RabbitMQ", zap.Error(err))
	}
	rt.Rabbit = rabbitConn
	rt.defer_(func() {
		if err := rabbitConn.Close(); err != nil {
			logger.Warn("rabbitmq close failed", zap.Error(err))
		}
	})
	logger.Info("RabbitMQ connection established")

	rt.Publisher = rabbitmq.NewPublisher(rabbitConn)
	// Every service declares every exchange, not just the ones it publishes
	// to. Declaring is idempotent, and it removes the startup race: a
	// consumer binding to an exchange whose owner has not booted yet would
	// otherwise fail and take the service down with it.
	if err := eventbus.DeclareModuleExchanges(rt.Publisher); err != nil {
		logger.Fatal("failed to declare module exchanges", zap.Error(err))
	}

	rt.Server = httpserver.NewServer(cfg, opts.Service)
	if rt.Pool != nil {
		rt.Server.RegisterHealthCheck("database", rt.Pool.Ping)
	}
	rt.Server.RegisterHealthCheck("rabbitmq", rabbitConn.Ping)
	if rt.Redis != nil {
		rt.Server.RegisterHealthCheck("redis", rt.Redis.Ping)
	}

	return rt
}

func (r *Runtime) initRedis(opts Options) {
	client, err := platformRedis.NewClient(r.Cfg.Redis.Addr, r.Cfg.Redis.Password, r.Cfg.Redis.DB)
	if err != nil {
		if opts.RedisFatal {
			logger.Fatal("failed to connect to Redis", zap.Error(err))
		}
		// Rate limiting degrades to open rather than refusing traffic.
		logger.Error("Redis unavailable, rate limiting disabled", zap.Error(err))
		return
	}
	r.Redis = client
	r.defer_(func() {
		if err := client.Close(); err != nil {
			logger.Warn("redis close failed", zap.Error(err))
		}
	})
	logger.Info("Redis connection established")

	// Every service, not just auth. JWTAuth skips the whole revocation block
	// when this global is nil, so a service that never sets it accepts tokens
	// that logout already revoked. Ten processes each have to set their own,
	// and the failure is silent — the token simply keeps working.
	platformMiddleware.SetBlacklistChecker(client)

	// Keys are namespaced per service, so a client that reuses one key across
	// two services cannot be served the wrong service's stored response.
	platformMiddleware.SetIdempotencyStore(client, opts.Service)

	// Write lock for demo ops editing mode.
	platformMiddleware.SetWriteLockStore(client)

	// The time machine's offset is shared through Redis. Without Redis the
	// service stays on the real clock, which is the safe side to fail on.
	clocksync.Start(r.Ctx, client)

	if !r.Cfg.RateLimit.Enabled {
		return
	}
	rl := platformMiddleware.RateLimitConfig{
		Enabled:        true,
		ServiceName:    sharedRateLimitBucket,
		IPLimit:        r.Cfg.RateLimit.IPLimit,
		IPWindow:       time.Duration(r.Cfg.RateLimit.IPWindowSecs) * time.Second,
		UserLimit:      r.Cfg.RateLimit.UserLimit,
		UserWindow:     time.Duration(r.Cfg.RateLimit.UserWindowSecs) * time.Second,
		EndpointLimits: r.endpointLimits(opts),
	}
	platformMiddleware.SetRateLimiter(platformMiddleware.NewRateLimiter(client, rl))
	logger.Info("rate limiter configured",
		zap.Int("ip_limit", r.Cfg.RateLimit.IPLimit),
		zap.Int("user_limit", r.Cfg.RateLimit.UserLimit),
	)
}

// InternalClient returns the transport for one target service, or stops the
// process. A missing URL is a deployment mistake, and discovering it on the
// first user request instead of at boot is strictly worse.
func (r *Runtime) InternalClient(target string) *client.Base {
	base, err := client.FromConfig(r.Cfg, target)
	if err != nil {
		logger.Fatal("internal client not configured", zap.Error(err))
	}
	return base
}

// endpointLimits builds the brute-force buckets. FailClosed is mandatory on
// all three: when Redis is unreachable a 503 beats letting an attacker try
// passwords without a limit.
func (r *Runtime) endpointLimits(opts Options) map[string]platformMiddleware.EndpointLimit {
	if !opts.LoginRateLimits {
		return nil
	}
	rl := r.Cfg.RateLimit
	return map[string]platformMiddleware.EndpointLimit{
		"login":    {Limit: rl.LoginLimit, Window: time.Duration(rl.LoginWindowSecs) * time.Second, FailClosed: true},
		"refresh":  {Limit: rl.RefreshLimit, Window: time.Duration(rl.RefreshWindowSecs) * time.Second, FailClosed: true},
		"password": {Limit: rl.PasswordLimit, Window: time.Duration(rl.PasswordWindowSecs) * time.Second, FailClosed: true},
	}
}

// DeclareQueues pre-declares the queues this service consumes, together with
// their bindings and dead-letter topology. Only the consumer declares its own
// queues; a publisher that declared them would keep a queue alive for a
// service that no longer exists.
func (r *Runtime) DeclareQueues(bindings []eventbus.DownstreamBinding) {
	if err := eventbus.DeclareDownstreamBindings(r.Publisher, bindings); err != nil {
		logger.Fatal("failed to declare downstream bindings", zap.Error(err))
	}
}

// StartOutbox runs the relay for this service's outbox table.
func (r *Runtime) StartOutbox(exchange string, store eventbus.OutboxStore) {
	go eventbus.NewOutboxWorker(
		r.service, exchange, store, r.Publisher,
		time.Duration(r.Cfg.Outbox.IntervalSeconds)*time.Second,
		utils.ClampToInt32(r.Cfg.Outbox.BatchSize),
	).Start(r.Ctx)
}

// StartRetention prunes this service's event tables on a schedule.
func (r *Runtime) StartRetention(store eventbus.RetentionStore) {
	go eventbus.NewRetentionWorker(
		r.service, store,
		r.Cfg.Timeout.ProcessedEventsRetentionDays,
		r.Cfg.Timeout.CleanupSchedulerIntervalHours,
	).Start(r.Ctx)
}

// Run mounts the service's module, serves, and blocks until SIGINT/SIGTERM.
// Shutdown cancels Ctx first so consumers and workers finish the message in
// flight, then drains HTTP — the reverse would leave half-processed events
// to be redelivered on restart.
func (r *Runtime) Run(module httpserver.Module) {
	r.Server.RegisterModules(module)
	r.Server.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down service")
	r.cancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := r.Server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}

	for i := len(r.closers) - 1; i >= 0; i-- {
		r.closers[i]()
	}
	logger.Info("service exited")
}

// defer_ registers a teardown step. They run in reverse order at the end of
// Run, which is where a main's defer statements would have run — main hands
// control to Run and never returns to its own defers.
func (r *Runtime) defer_(fn func()) {
	r.closers = append(r.closers, fn)
}
