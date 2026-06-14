package dto

import (
	"time"

	"github.com/google/uuid"
)

// BaseEvent contains common event fields
type BaseEvent struct {
	EventID   uuid.UUID   `json:"event_id"`
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      any `json:"data"`
}

// =======================================
// CONSUMED EVENTS
// =======================================

// StudentCreatedEventData is consumed from student service
type StudentCreatedEventData struct {
	StudentID     uuid.UUID `json:"id"`
	StudentNumber string    `json:"student_number"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Email         string    `json:"email"`
	Department    string    `json:"department"`
}

// StudentUpdatedEventData is consumed from student service
type StudentUpdatedEventData struct {
	StudentID     uuid.UUID `json:"id"`
	StudentNumber string    `json:"student_number"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Email         string    `json:"email"`
	Department    string    `json:"department"`
}

// StudentDeactivatedEventData is consumed from student service
type StudentDeactivatedEventData struct {
	StudentID uuid.UUID `json:"id"`
}

// ScheduleSessionInfo represents a schedule session from catalog events
type ScheduleSessionInfo struct {
	SessionType string `json:"session_type"`
}

// CourseSemesterCreatedEvent is consumed from course catalog service
// Field names must match catalog service's flat event payload
type CourseSemesterCreatedEventData struct {
	SemesterCourseID   uuid.UUID            `json:"semester_course_id"`
	CourseCode         string               `json:"course_code"`
	CourseName         string               `json:"course_name"`
	Credits            int16                `json:"credits"`
	Semester           string               `json:"semester"`
	Department         string               `json:"department"`
	InstructorID       uuid.UUID            `json:"instructor_id"`
	InstructorFullname string               `json:"instructor_fullname"`
	ScheduleSessions   []ScheduleSessionInfo `json:"schedule_sessions"`
}

// EnrollmentProgramApprovedEventData is consumed from enrollment service
// Wrapped in BaseEvent: { event_id, event_type, timestamp, data: { ... } }
type EnrollmentProgramApprovedEventData struct {
	ProgramID  uuid.UUID   `json:"program_id"`
	StudentID  uuid.UUID   `json:"student_id"`
	Semester   string      `json:"semester"`
	CourseIDs  []uuid.UUID `json:"course_ids"`
	ApprovedBy uuid.UUID   `json:"approved_by"`
}

// =======================================
// PUBLISHED EVENTS
// =======================================

// AttendanceFailedTypeDetail contains attendance detail for a session type
type AttendanceFailedTypeDetail struct {
	TotalSessions int `json:"total_sessions"`
	PresentCount  int `json:"present_count"`
	AbsentCount   int `json:"absent_count"`
	MinRequired   int `json:"min_required"`
}

// AttendanceSemesterFailedEventData is published when student fails due to attendance
type AttendanceSemesterFailedEventData struct {
	StudentID     uuid.UUID                    `json:"student_id"`
	StudentNumber string                       `json:"student_number"`
	StudentEmail  string                       `json:"student_email"`
	CourseID      uuid.UUID                    `json:"course_id"`
	CourseCode    string                       `json:"course_code"`
	CourseName    string                       `json:"course_name"`
	Semester      string                       `json:"semester"`
	TotalWeeks    int16                        `json:"total_weeks"`
	FailedType    string                       `json:"failed_type"` // "theory", "lab", or "both"
	Theory        *AttendanceFailedTypeDetail  `json:"theory,omitempty"`
	Lab           *AttendanceFailedTypeDetail  `json:"lab,omitempty"`
}
