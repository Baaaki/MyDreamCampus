package worker

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/service"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"go.uber.org/zap"
)

type BufferFlusher struct {
	attendanceRepo *repository.AttendanceRepository
	redisService   *service.RedisService
}

func NewBufferFlusher(
	attendanceRepo *repository.AttendanceRepository,
	redisService   *service.RedisService,
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
	log := logger.WithContextAndFields(ctx,
		zap.String("worker", "BufferFlusher"),
		zap.String("method", "flushSingleBuffer"),
	)

	// Extract sessionID from key (format: attendance:buffer:SESSION_ID)
	parts := strings.Split(bufferKey, ":")
	if len(parts) != 3 {
		return nil
	}
	sessionID := parts[2]

	// Get all records from this buffer as map[studentID]jsonData
	bufferData, err := w.redisService.GetBuffer(ctx, sessionID)
	if err != nil {
		return err
	}

	if len(bufferData) == 0 {
		return nil
	}

	log.Info("flushing buffer",
		zap.String("buffer", bufferKey),
		zap.Int("record_count", len(bufferData)),
	)

	// Parse buffered JSON into params. Unmarshal failures are dropped from both
	// the batch and the buffer — replaying a corrupt record on every tick is pointless.
	records := make([]db.CreateAttendanceRecordQRParams, 0, len(bufferData))
	parsedFields := make([]string, 0, len(bufferData))
	corruptFields := make([]string, 0)
	for studentID, jsonData := range bufferData {
		var record db.CreateAttendanceRecordQRParams
		if err := json.Unmarshal([]byte(jsonData), &record); err != nil {
			log.Error("failed to unmarshal buffered record",
				zap.String("student_id", studentID),
				zap.Error(err),
			)
			corruptFields = append(corruptFields, studentID)
			continue
		}
		records = append(records, record)
		parsedFields = append(parsedFields, studentID)
	}

	// Single batch insert — all-or-nothing. ON CONFLICT DO NOTHING makes retries safe.
	if len(records) > 0 {
		if err := w.attendanceRepo.BatchCreateAttendanceRecordsQR(ctx, records); err != nil {
			log.Error("failed to batch insert buffered records",
				zap.String("buffer", bufferKey),
				zap.Int("record_count", len(records)),
				zap.Error(err),
			)
			// Drop corrupt fields even on insert failure so we don't replay them forever.
			if len(corruptFields) > 0 {
				_ = w.redisService.HDelBufferFields(ctx, sessionID, corruptFields)
			}
			return err
		}
	}

	// Delete only the fields we successfully processed — anything added to the
	// buffer between GetBuffer and now stays for the next tick.
	toDelete := append(parsedFields, corruptFields...)
	if len(toDelete) > 0 {
		if err := w.redisService.HDelBufferFields(ctx, sessionID, toDelete); err != nil {
			log.Error("failed to hdel flushed fields",
				zap.String("buffer", bufferKey),
				zap.Error(err),
			)
		}
	}

	log.Info("buffer flushed successfully",
		zap.String("buffer", bufferKey),
		zap.Int("success_count", len(records)),
		zap.Int("corrupt_count", len(corruptFields)),
	)

	return nil
}
