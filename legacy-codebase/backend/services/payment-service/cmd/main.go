package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/baaaki/mydreamcampus/payment-service/config"
	"github.com/baaaki/mydreamcampus/payment-service/internal/server"
	pb "github.com/baaaki/mydreamcampus/payment-service/proto"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	logger.MustInit(cfg.Server.Environment)
	log := logger.Log
	defer logger.Sync()

	log.Info("starting payment service (gRPC)",
		zap.String("environment", cfg.Server.Environment),
		zap.String("port", cfg.Server.Port),
	)

	// Create TCP listener
	lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register payment service
	paymentServer := server.NewPaymentServer(log)
	pb.RegisterPaymentServiceServer(grpcServer, paymentServer)

	// Enable reflection for debugging (grpcurl, etc.)
	reflection.Register(grpcServer)

	// Start gRPC server in goroutine
	go func() {
		log.Info("gRPC server listening", zap.String("address", lis.Addr().String()))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gRPC server...")

	// Graceful shutdown
	grpcServer.GracefulStop()

	log.Info("server exited")
}
