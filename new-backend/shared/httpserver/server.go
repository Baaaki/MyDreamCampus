package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/shared/config"
	platformHandler "github.com/baaaki/mydreamcampus/shared/platform/handler"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Module is implemented by every business module. The HTTP server walks
// the registered modules at startup and lets each one wire its own routes
// onto an /api/<module-name> subgroup.
type Module interface {
	Name() string
	RegisterRoutes(rg *gin.RouterGroup)
}

// PublicRoutesProvider is an optional companion interface. Modules that
// need to expose routes outside the /api/<name> group (e.g. public
// teacher pages, anonymous lookups) implement it to receive the root
// engine. The server invokes RegisterPublicRoutes after RegisterRoutes.
type PublicRoutesProvider interface {
	RegisterPublicRoutes(r *gin.Engine)
}

// Server bundles a Gin router around the service config plus dependency
// health checks. main.go constructs it once, registers its module, then
// calls Run/Shutdown.
type Server struct {
	cfg          *config.Config
	service      string
	router       *gin.Engine
	httpServer   *http.Server
	healthChecks map[string]platformHandler.HealthCheck
}

// NewServer builds the router with the middleware chain every service runs.
// service names the binary in /health and /ready output.
func NewServer(cfg *config.Config, service string) *Server {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// Gin trusts every peer by default, so ClientIP() returned whatever the
	// client wrote into X-Forwarded-For — spoofable, which let anyone pick
	// a fresh rate-limit bucket per request. With the peer list set, the
	// header is honoured only as far as it was written by a known proxy.
	if err := r.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		logger.Fatal("invalid TRUSTED_PROXIES", zap.Error(err))
	}
	r.Use(platformMiddleware.Recovery())
	r.Use(platformMiddleware.SecurityHeaders())
	r.Use(platformMiddleware.CORS())
	r.Use(platformMiddleware.BodySizeLimit(platformMiddleware.DefaultMaxBodyBytes))
	r.Use(platformMiddleware.RequestLogger())
	r.Use(platformMiddleware.IPRateLimit())
	r.Use(platformMiddleware.SetCSRFToken(cfg.Server.Environment == "production"))

	return &Server{
		cfg:          cfg,
		service:      service,
		router:       r,
		healthChecks: make(map[string]platformHandler.HealthCheck),
	}
}

// Engine exposes the underlying Gin router for tests and advanced wiring.
func (s *Server) Engine() *gin.Engine {
	return s.router
}

// RegisterHealthCheck adds a dependency check that /ready will run.
// Pass platform pings (DB Ping, RabbitMQ Ping, Redis Ping) here from main.go.
func (s *Server) RegisterHealthCheck(name string, check platformHandler.HealthCheck) {
	s.healthChecks[name] = check
}

// RegisterModules mounts each module under /api/<name> and finalises the
// /health and /ready routes. Call exactly once after all modules and health
// checks are registered.
func (s *Server) RegisterModules(modules ...Module) {
	api := s.router.Group("/api")
	for _, m := range modules {
		group := api.Group("/" + m.Name())
		m.RegisterRoutes(group)
		if pub, ok := m.(PublicRoutesProvider); ok {
			pub.RegisterPublicRoutes(s.router)
		}
		logger.Info("module registered", zap.String("module", m.Name()))
	}

	s.router.GET("/health", platformHandler.LivenessHandler(s.service))
	s.router.GET("/ready", platformHandler.ReadinessHandler(s.service, s.healthChecks))
}

// Run starts ListenAndServe in a goroutine; the call returns immediately.
// Caller is responsible for invoking Shutdown on signal.
func (s *Server) Run() {
	// Without explicit timeouts net/http keeps slow connections open
	// forever (slowloris). Read/Write derive from REQUEST_TIMEOUT_SECONDS.
	requestTimeout := time.Duration(s.cfg.Timeout.RequestTimeoutSeconds) * time.Second
	if requestTimeout <= 0 {
		requestTimeout = 10 * time.Second
	}
	s.httpServer = &http.Server{
		Addr:              ":" + s.cfg.Server.Port,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       requestTimeout,
		WriteTimeout:      requestTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		logger.Info("http server starting", zap.String("port", s.cfg.Server.Port))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server failed", zap.Error(err))
		}
	}()
}

// Shutdown gives in-flight requests up to ShutdownTimeoutSeconds to drain.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	timeout := time.Duration(s.cfg.Timeout.ShutdownTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
