package service

import (
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	"github.com/baaaki/mydreamcampus/attendance/internal/dto"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func heldSession(week int16, sessionType db.SessionTypeEnum, date string, active bool) db.GetSessionsByCourseRow {
	day, _ := time.Parse("2006-01-02", date)
	return db.GetSessionsByCourseRow{
		WeekNumber:  week,
		SessionType: sessionType,
		SessionDate: pgtype.Date{Time: day, Valid: true},
		IsActive:    pgtype.Bool{Bool: active, Valid: true},
	}
}

func attendedSession(week int16, sessionType db.SessionTypeEnum) db.GetStudentAttendanceByCourseRow {
	return db.GetStudentAttendanceByCourseRow{WeekNumber: week, SessionType: sessionType}
}

func TestMissedSessions_SessionsWithoutRecord_ListedInOrder(t *testing.T) {
	sessions := []db.GetSessionsByCourseRow{
		heldSession(1, db.AttendanceSessionTypeEnumTheory, "2026-09-28", false),
		heldSession(1, db.AttendanceSessionTypeEnumLab, "2026-10-01", false),
		heldSession(2, db.AttendanceSessionTypeEnumTheory, "2026-10-05", false),
	}
	attended := []db.GetStudentAttendanceByCourseRow{
		attendedSession(1, db.AttendanceSessionTypeEnumTheory),
	}

	assert.Equal(t, []dto.MissedSession{
		{Week: 1, SessionType: "lab", Date: "2026-10-01"},
		{Week: 2, SessionType: "theory", Date: "2026-10-05"},
	}, missedSessions(sessions, attended))
}

func TestMissedSessions_OpenSession_NotMissedYet(t *testing.T) {
	sessions := []db.GetSessionsByCourseRow{
		heldSession(3, db.AttendanceSessionTypeEnumTheory, "2026-10-12", true),
	}

	assert.Empty(t, missedSessions(sessions, nil))
}

func TestMissedSessions_NoSessions_EmptyListNotNull(t *testing.T) {
	missed := missedSessions(nil, nil)

	assert.NotNil(t, missed)
	assert.Empty(t, missed)
}
