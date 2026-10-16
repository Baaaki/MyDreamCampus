package service

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"go.uber.org/zap"
)

// ScanRecordWriter and ScanBuffer are what draining a session's scan buffer
// needs: the batch insert, and the Redis hash QR scans wait in.
type ScanRecordWriter interface {
	BatchCreateAttendanceRecordsQR(ctx context.Context, records []db.CreateAttendanceRecordQRParams) error
}

type ScanBuffer interface {
	GetBuffer(ctx context.Context, sessionID string) (map[string]string, error)
	HDelBufferFields(ctx context.Context, sessionID string, studentIDs []string) error
}

// FlushSessionBuffer moves one session's buffered QR scans into the
// database. The BufferFlusher worker runs it for every buffer on its tick;
// CloseSession and FinalizeAttendance run it for their own sessions, so they
// never act on a roll that is still partly in Redis.
func FlushSessionBuffer(ctx context.Context, records ScanRecordWriter, buffer ScanBuffer, sessionID string) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("method", "FlushSessionBuffer"),
		zap.String("session_id", sessionID),
	)

	// Get all records from this buffer as map[studentID]jsonData
	bufferData, err := buffer.GetBuffer(ctx, sessionID)
	if err != nil {
		return err
	}

	if len(bufferData) == 0 {
		return nil
	}

	log.Info("flushing buffer", zap.Int("record_count", len(bufferData)))

	// Parse buffered JSON into params. Unmarshal failures are dropped from both
	// the batch and the buffer — replaying a corrupt record on every tick is pointless.
	parsed := make([]db.CreateAttendanceRecordQRParams, 0, len(bufferData))
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
		parsed = append(parsed, record)
		parsedFields = append(parsedFields, studentID)
	}

	// Single batch insert — all-or-nothing. ON CONFLICT DO NOTHING makes retries safe.
	if len(parsed) > 0 {
		if err := records.BatchCreateAttendanceRecordsQR(ctx, parsed); err != nil {
			log.Error("failed to batch insert buffered records",
				zap.Int("record_count", len(parsed)),
				zap.Error(err),
			)
			// Drop corrupt fields even on insert failure so we don't replay them forever.
			if len(corruptFields) > 0 {
				_ = buffer.HDelBufferFields(ctx, sessionID, corruptFields)
			}
			return err
		}
	}

	// Delete only the fields we successfully processed — anything added to the
	// buffer between GetBuffer and now stays for the next flush. A failed
	// HDEL is not a failed flush: the rows are in, and the next flush
	// re-inserts them as no-ops before deleting them.
	toDelete := append(parsedFields, corruptFields...)
	if err := buffer.HDelBufferFields(ctx, sessionID, toDelete); err != nil {
		log.Error("failed to hdel flushed fields", zap.Error(err))
	}

	log.Info("buffer flushed successfully",
		zap.Int("success_count", len(parsed)),
		zap.Int("corrupt_count", len(corruptFields)),
	)

	return nil
}
