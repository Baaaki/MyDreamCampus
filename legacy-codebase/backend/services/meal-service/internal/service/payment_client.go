package service

import (
	"context"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/meal-service/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/meal-service/internal/errors"
	pb "github.com/baaaki/mydreamcampus/meal-service/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PaymentClient struct {
	grpcClient pb.PaymentServiceClient
	conn       *grpc.ClientConn
	logger     *zap.Logger
}

func NewPaymentClient(address string, logger *zap.Logger) (*PaymentClient, error) {
	// Create gRPC connection with retry and timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		logger.Error("failed to connect to payment service", zap.Error(err), zap.String("address", address))
		return nil, fmt.Errorf("failed to connect to payment service: %w", err)
	}

	client := pb.NewPaymentServiceClient(conn)

	logger.Info("connected to payment service via gRPC", zap.String("address", address))

	return &PaymentClient{
		grpcClient: client,
		conn:       conn,
		logger:     logger,
	}, nil
}

// Close closes the gRPC connection
func (c *PaymentClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// InitiatePayment initiates payment for reservation(s) via gRPC
func (c *PaymentClient) InitiatePayment(ctx context.Context, req dto.InitiatePaymentRequest) (*dto.InitiatePaymentResponse, error) {
	c.logger.Info("initiating payment via gRPC",
		zap.String("reference_id", req.ReferenceID),
		zap.Float64("amount", req.Amount),
		zap.String("currency", req.Currency),
	)

	// Call gRPC service
	grpcReq := &pb.InitiatePaymentRequest{
		ReferenceId: req.ReferenceID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		StudentId:   req.StudentID,
	}

	resp, err := c.grpcClient.InitiatePayment(ctx, grpcReq)
	if err != nil {
		c.logger.Error("gRPC payment initiation failed", zap.Error(err))
		return nil, fmt.Errorf("%w: %v", serviceErrors.ErrPaymentFailed, err)
	}

	if !resp.Success {
		c.logger.Error("payment initiation returned failure", zap.String("error", resp.ErrorMessage))
		return nil, fmt.Errorf("%w: %s", serviceErrors.ErrPaymentFailed, resp.ErrorMessage)
	}

	c.logger.Info("payment initiated successfully",
		zap.String("payment_id", resp.PaymentId),
		zap.String("payment_url", resp.PaymentUrl),
	)

	return &dto.InitiatePaymentResponse{
		PaymentID:  resp.PaymentId,
		PaymentURL: resp.PaymentUrl,
		Amount:     resp.Amount,
		Currency:   resp.Currency,
		ExpiresAt:  resp.ExpiresAt,
	}, nil
}

// RequestRefund requests refund for a cancelled reservation via gRPC
func (c *PaymentClient) RequestRefund(ctx context.Context, req dto.RefundRequest) (*dto.RefundResponse, error) {
	c.logger.Info("requesting refund via gRPC",
		zap.String("reference_id", req.ReferenceID),
		zap.Float64("amount", req.Amount),
		zap.String("currency", req.Currency),
	)

	// Call gRPC service
	grpcReq := &pb.RefundRequest{
		ReferenceId: req.ReferenceID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Reason:      req.Reason,
	}

	resp, err := c.grpcClient.RequestRefund(ctx, grpcReq)
	if err != nil {
		c.logger.Error("gRPC refund request failed", zap.Error(err))
		return nil, fmt.Errorf("%w: %v", serviceErrors.ErrRefundFailed, err)
	}

	if !resp.Success {
		c.logger.Error("refund request returned failure", zap.String("message", resp.Message))
		return nil, fmt.Errorf("%w: %s", serviceErrors.ErrRefundFailed, resp.Message)
	}

	c.logger.Info("refund completed successfully",
		zap.String("refund_id", resp.RefundId),
		zap.String("status", resp.Status),
	)

	return &dto.RefundResponse{
		RefundID: resp.RefundId,
		Amount:   resp.Amount,
		Currency: resp.Currency,
		Status:   resp.Status,
		Message:  resp.Message,
	}, nil
}
