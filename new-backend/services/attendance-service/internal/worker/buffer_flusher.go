package worker

import (
	"context"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/attendance/internal/repository"
	"github.com/baaaki/mydreamcampus/attendance/internal/service"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"go.uber.org/zap"
)

type BufferFlusher struct {
	attendanceRepo *repository.AttendanceRepository
	redisService   *service.RedisService
}

func NewBufferFlusher(
	attendanceRepo *repository.AttendanceRepository,
	redisService *service.RedisService,
) *BufferFlusher {
	return &BufferFlusher{
		attendanceRepo: attendanceRepo,
		redisService:   redisService,
	}
}

func (w *BufferFlusher) Start(ctx context.Context) {
	log := logger.WithContextAndFields(ctx, zap.String("worker", "BufferFlusher"))

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Info("Buffer flusher started")

	for {
		select {
		case <-ctx.Done():
			log.Info("Buffer flusher stopped")
			return
		case <-ticker.C:
			if err := w.flushBuffers(ctx); err != nil {
				log.Error("failed to flush buffers", zap.Error(err))
			}
		}
	}
}

func (w *BufferFlusher) flushBuffers(ctx context.Context) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "BufferFlusher"),
		zap.String("method", "flushBuffers"),
	)

	// Get all session buffer keys from Redis (pattern: attendance:buffer:*)
	bufferKeys, err := w.redisService.GetAllBufferKeys(ctx)
	if err != nil {
		log.Error("failed to get buffer keys", zap.Error(err))
		return err
	}

	if len(bufferKeys) == 0 {
		log.Debug("no buffers to flush")
		return nil
	}

	log.Info("flushing buffers", zap.Int("buffer_count", len(bufferKeys)))

	// Process each buffer
	for _, key := range bufferKeys {
		if err := w.flushSingleBuffer(ctx, key); err != nil {
			log.Error("failed to flush buffer",
				zap.String("key", key),
				zap.Error(err),
			)
			// Continue to next buffer on error
			continue
		}
	}

	log.Debug("buffer flush cycle completed")
	return nil
}

func (w *BufferFlusher) flushSingleBuffer(ctx context.Context, bufferKey string) error {
	// Extract sessionID from key (format: attendance:buffer:SESSION_ID)
	parts := strings.Split(bufferKey, ":")
	if len(parts) != 3 {
		return nil
	}
	return service.FlushSessionBuffer(ctx, w.attendanceRepo, w.redisService, parts[2])
}
