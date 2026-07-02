package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baaaki/mydreamcampus/notification/config"
	"github.com/baaaki/mydreamcampus/notification/internal/consumer"
	"github.com/baaaki/mydreamcampus/notification/internal/delivery/email"
	"github.com/baaaki/mydreamcampus/notification/internal/delivery/push"
	"github.com/baaaki/mydreamcampus/notification/internal/repository"
	"github.com/baaaki/mydreamcampus/notification/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// DB
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// RabbitMQ
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		logger.Fatal("failed to connect to RabbitMQ", zap.Error(err))
	}
	defer func() { _ = conn.Close() }()

	ch, err := conn.Channel()
	if err != nil {
		logger.Fatal("failed to open channel", zap.Error(err))
	}
	defer func() { _ = ch.Close() }()

	// Topology setup
	if err := consumer.SetupTopology(ch); err != nil {
		logger.Fatal("topology setup failed", zap.Error(err))
	}

	// Adapters
	smtp := email.NewSMTPSender(cfg.SMTP)
	pushSender := push.New(logger)

	// Repository + Service
	repo := repository.New(pool)
	svc, err := service.New(repo, smtp, pushSender, logger, cfg)
	if err != nil {
		logger.Fatal("failed to create service", zap.Error(err))
	}

	// Consumer
	cons := consumer.New(ch, svc, repo, logger)
	go cons.Start(ctx)

	// Health endpoint
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		})
		healthServer := &http.Server{
			Addr:              ":9090",
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
		logger.Info("Starting health server on :9090")
		if err := healthServer.ListenAndServe(); err != nil {
			logger.Error("health server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down notification service")
}
