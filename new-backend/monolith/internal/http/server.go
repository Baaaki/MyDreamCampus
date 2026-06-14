package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/config"
	platformHandler "github.com/baaaki/mydreamcampus/monolith/internal/platform/handler"
	platformMiddleware "github.com/baaaki/mydreamcampus/monolith/internal/platform/middleware"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
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

// Server bundles a Gin router around the monolith config plus dependency
// health checks. main.go constructs it once, registers modules, then calls
// Run/Shutdown.
type Server struct {
	cfg          *config.Config
	router       *gin.Engine
	httpServer   *http.Server
	healthChecks map[string]platformHandler.HealthCheck
}

func NewServer(cfg *config.Config) *Server {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(platformMiddleware.Recovery())
	r.Use(platformMiddleware.SecurityHeaders())
	r.Use(platformMiddleware.CORS())
	r.Use(platformMiddleware.RequestLogger())
	r.Use(platformMiddleware.IPRateLimit())
	r.Use(platformMiddleware.SetCSRFToken(cfg.Server.Environment == "production"))

	return &Server{
		cfg:          cfg,
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
// /health, /ready, and SPA fallback routes. Call exactly once after all
// modules and health checks are registered.
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

	s.router.GET("/health", platformHandler.LivenessHandler("monolith"))
	s.router.GET("/ready", platformHandler.ReadinessHandler("monolith", s.healthChecks))

	s.installFrontendFallback()
}

// installFrontendFallback wires SPA static serving for production. In dev
// the Vite proxy hits /api/* directly so this is left disabled — see
// FRONTEND_STATIC_ENABLED in config.
func (s *Server) installFrontendFallback() {
	if !s.cfg.Frontend.Enabled {
		return
	}

	dir := s.cfg.Frontend.StaticDir
	if dir == "" {
		logger.Warn("frontend serving enabled but FRONTEND_STATIC_DIR is empty")
		return
	}

	assets := filepath.Join(dir, "assets")
	if _, err := os.Stat(assets); err == nil {
		s.router.Static("/assets", assets)
	}
	if _, err := os.Stat(filepath.Join(dir, "favicon.ico")); err == nil {
		s.router.StaticFile("/favicon.ico", filepath.Join(dir, "favicon.ico"))
	}

	indexFile := filepath.Join(dir, "index.html")
	s.router.NoRoute(func(c *gin.Context) {
		// Do not swallow API 404s — only fall back to SPA for non-/api paths.
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}
		c.File(indexFile)
	})
}

// Run starts ListenAndServe in a goroutine; the call returns immediately.
// Caller is responsible for invoking Shutdown on signal.
func (s *Server) Run() {
	s.httpServer = &http.Server{
		Addr:    ":" + s.cfg.Server.Port,
		Handler: s.router,
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
