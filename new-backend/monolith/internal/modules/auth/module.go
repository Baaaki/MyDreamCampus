// Package auth wires the auth module's repositories, services and
// handlers behind a single Module struct that main.go consumes.
//
// The module owns the auth schema (users, sessions, processed_events).
// It implements monolithHTTP.Module so it can be registered onto the
// shared Gin router under /api/auth, and exposes Bootstrap for the
// startup-time work that has to run after dependency injection
// (admin seed, cleanup scheduler, event consumer) — see plan section 3.
package auth

import (
	"context"

	"github.com/baaaki/mydreamcampus/monolith/config"
	"github.com/baaaki/mydreamcampus/monolith/internal/eventbus"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/handler"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/auth/worker"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/monolith/internal/platform/middleware"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
	platformRedis "github.com/baaaki/mydreamcampus/monolith/internal/platform/redis"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Module is the wiring root for the auth module.
type Module struct {
	cfg         *config.Config
	pool        *pgxpool.Pool
	redisClient *platformRedis.ClientWrapper

	authRepo    *repository.AuthRepository
	sessionRepo *repository.SessionRepository
	eventRepo   *repository.EventRepository
	outboxRepo  *repository.OutboxRepository
	outboxStore *repository.OutboxStore

	authService  *service.AuthService
	eventService *service.EventService

	handler  *handler.AuthHandler
	consumer *worker.EventConsumer
}

// New constructs the auth module. Caller (main.go) provides the shared
// infrastructure — DB pool, Redis client, RabbitMQ connection — so the
// module never opens its own connections.
func New(
	cfg *config.Config,
	pool *pgxpool.Pool,
	redisClient *platformRedis.ClientWrapper,
	rabbitConn *rabbitmq.Connection,
) *Module {
	authRepo := repository.NewAuthRepository(pool)
	sessionRepo := repository.NewSessionRepository(pool)
	eventRepo := repository.NewEventRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)

	authService := service.NewAuthService(authRepo, sessionRepo, eventRepo, redisClient, cfg)
	eventService := service.NewEventService(authRepo, eventRepo, pool)

	authHandler := handler.NewAuthHandler(authService, cfg)

	consumer := rabbitmq.NewConsumer(rabbitConn)
	eventConsumer := worker.NewEventConsumer(consumer, eventService)

	return &Module{
		cfg:          cfg,
		pool:         pool,
		redisClient:  redisClient,
		authRepo:     authRepo,
		sessionRepo:  sessionRepo,
		eventRepo:    eventRepo,
		outboxRepo:   outboxRepo,
		outboxStore:  repository.NewOutboxStore(outboxRepo),
		authService:  authService,
		eventService: eventService,
		handler:      authHandler,
		consumer:     eventConsumer,
	}
}

// Name implements monolithHTTP.Module — used as the URL prefix segment.
func (m *Module) Name() string { return "auth" }

// OutboxStore exposes the eventbus.OutboxStore for the per-module outbox worker
func (m *Module) OutboxStore() eventbus.OutboxStore { return m.outboxStore }

// RegisterRoutes implements monolithHTTP.Module. Routes are mounted under
// /api/auth by the server. Auth-specific middleware (JWTAuth, CSRF,
// per-endpoint rate limits) is wired at the route-group level here.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	// Public — no auth required.
	rg.POST("/login", platformMiddleware.EndpointRateLimit("login"), m.handler.Login)
	rg.POST("/refresh", platformMiddleware.EndpointRateLimit("refresh"), m.handler.RefreshToken)
	rg.POST("/request-password-reset", platformMiddleware.EndpointRateLimit("password"), m.handler.RequestPasswordReset)

	// Protected — JWT + CSRF + per-user rate limit.
	protected := rg.Group("")
	protected.Use(platformMiddleware.JWTAuth())
	protected.Use(platformMiddleware.CSRFProtection())
	protected.Use(platformMiddleware.UserRateLimit())
	{
		protected.POST("/logout", m.handler.Logout)
		protected.POST("/logout-all", m.handler.LogoutAll)
		// Password change re-runs JWTAuth in fail-closed mode so a stale
		// or blacklisted token cannot slip through if Redis is unreachable.
		protected.POST("/change-password",
			platformMiddleware.JWTAuth(platformMiddleware.WithFailClosed()),
			platformMiddleware.EndpointRateLimit("password"),
			m.handler.ChangePassword,
		)
		protected.GET("/sessions", m.handler.GetSessions)
		protected.DELETE("/sessions/:id", m.handler.DeleteSession)
		protected.GET("/verify", m.handler.Verify)
	}
}

// Bootstrap runs auth's startup-time work: register the JWT middleware's
// blacklist checker, seed the initial admin user, start the cleanup
// scheduler, and begin consuming staff/student events from RabbitMQ.
//
// Cross-module event consumption stays on RabbitMQ for now; once staff
// and student modules migrate (Faz 2) we can replace the RabbitMQ
// hop with an in-process subscriber per plan section 8.
func (m *Module) Bootstrap(ctx context.Context) error {
	platformMiddleware.SetBlacklistChecker(m.redisClient)

	if err := m.authService.SeedAdmin(ctx); err != nil {
		logger.Error("admin seed failed", zap.Error(err))
		// Non-fatal: admin may already exist.
	}

	m.authService.StartCleanupScheduler(ctx)

	if err := m.consumer.Start(ctx); err != nil {
		return err
	}
	return nil
}
