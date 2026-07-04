package worker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
)

// The Redis buffer is a serialization boundary: ScanQR marshals
// db.CreateAttendanceRecordQRParams into the buffer hash and the flusher
// unmarshals it back. pgtype fields must survive that roundtrip byte-exact,
// otherwise scans silently lose data between Redis and Postgres.
func TestBufferRecord_JSONRoundtrip_PreservesAllFields(t *testing.T) {
	scannedAt := time.Date(2026, 7, 4, 10, 30, 0, 0, time.UTC)
	original := db.CreateAttendanceRecordQRParams{
		SessionID:   utils.UUIDToPgUUID(uuid.New()),
		StudentID:   utils.UUIDToPgUUID(uuid.New()),
		CourseID:    utils.UUIDToPgUUID(uuid.New()),
		Semester:    "2025-2026-Fall",
		WeekNumber:  7,
		ScannedAt:   utils.TimeToPgTimestamp(scannedAt),
		SessionType: db.AttendanceSessionTypeEnumLab,
	}

	buf, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded db.CreateAttendanceRecordQRParams
	if err := json.Unmarshal(buf, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID mismatch: %v != %v", decoded.SessionID, original.SessionID)
	}
	if decoded.StudentID != original.StudentID {
		t.Errorf("StudentID mismatch: %v != %v", decoded.StudentID, original.StudentID)
	}
	if decoded.CourseID != original.CourseID {
		t.Errorf("CourseID mismatch: %v != %v", decoded.CourseID, original.CourseID)
	}
	if decoded.Semester != original.Semester {
		t.Errorf("Semester mismatch: %q != %q", decoded.Semester, original.Semester)
	}
	if decoded.WeekNumber != original.WeekNumber {
		t.Errorf("WeekNumber mismatch: %d != %d", decoded.WeekNumber, original.WeekNumber)
	}
	if decoded.SessionType != original.SessionType {
		t.Errorf("SessionType mismatch: %q != %q", decoded.SessionType, original.SessionType)
	}
	if !decoded.ScannedAt.Valid {
		t.Fatal("ScannedAt lost validity in roundtrip")
	}
	if !decoded.ScannedAt.Time.Equal(scannedAt) {
		t.Errorf("ScannedAt mismatch: %v != %v", decoded.ScannedAt.Time, scannedAt)
	}
	// QrTimestamp is never set by ScanQR; it must stay NULL, not become 0.
	if decoded.QrTimestamp.Valid {
		t.Errorf("QrTimestamp should stay invalid (NULL), got %+v", decoded.QrTimestamp)
	}
}
