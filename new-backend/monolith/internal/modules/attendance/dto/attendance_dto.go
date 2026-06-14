package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateSessionRequest is the request to create attendance session
type CreateSessionRequest struct {
	CourseID        uuid.UUID `json:"course_id" binding:"required"`
	WeekNumber      int16     `json:"week_number" binding:"required,min=1,max=14"`
	DurationMinutes int       `json:"duration_minutes" binding:"required,min=5,max=120"`
	SessionType     string    `json:"session_type" binding:"required,oneof=theory lab"`
}

// CreateSessionResponse is the response for created session
type CreateSessionResponse struct {
	SessionID            uuid.UUID `json:"session_id"`
	CourseID             uuid.UUID `json:"course_id"`
	CourseCode           string    `json:"course_code"`
	CourseName           string    `json:"course_name"`
	WeekNumber           int16     `json:"week_number"`
	SessionType          string    `json:"session_type"`
	SessionDate          string    `json:"session_date"`
	StartedAt            time.Time `json:"started_at"`
	ExpiresAt            time.Time `json:"expires_at"`
	EnrolledStudentCount int       `json:"enrolled_student_count"`
}

// QRPayload is the data embedded in QR code
type QRPayload struct {
	SessionID string `json:"sid"`
	Signature string `json:"sig"`
}

// GetQRResponse is the response for QR code data
type GetQRResponse struct {
	SessionID  uuid.UUID `json:"session_id"`
	QRPayload  QRPayload `json:"qr_payload"`
	ValidUntil time.Time `json:"valid_until"`
}

// ScanQRRequest is the request for QR code scanning
type ScanQRRequest struct {
	QRPayload QRPayload `json:"qr_payload" binding:"required"`
}

// ScanQRResponse is the response for successful scan
type ScanQRResponse struct {
	Message     string    `json:"message"`
	CourseCode  string    `json:"course_code"`
	CourseName  string    `json:"course_name"`
	WeekNumber  int16     `json:"week_number"`
	SessionType string    `json:"session_type"`
	MarkedAt    time.Time `json:"marked_at"`
}

// ManualAttendanceRequest is the request for manual attendance entry
type ManualAttendanceRequest struct {
	StudentID uuid.UUID `json:"student_id" binding:"required"`
	Note      string    `json:"note"`
}

// ManualAttendanceResponse is the response for manual attendance
type ManualAttendanceResponse struct {
	ID            uuid.UUID  `json:"id"`
	SessionID     uuid.UUID  `json:"session_id"`
	StudentID     uuid.UUID  `json:"student_id"`
	StudentNumber string     `json:"student_number"`
	StudentName   string     `json:"student_name"`
	MarkedVia     string     `json:"marked_via"`
	Note          *string    `json:"note,omitempty"`
	MarkedAt      *time.Time `json:"marked_at"`
}

// CloseSessionResponse is the response for closing session
type CloseSessionResponse struct {
	SessionID uuid.UUID      `json:"session_id"`
	ClosedAt  time.Time      `json:"closed_at"`
	Summary   SessionSummary `json:"summary"`
}

// SessionSummary contains attendance summary for a session
type SessionSummary struct {
	TotalEnrolled int `json:"total_enrolled"`
	PresentCount  int `json:"present_count"`
	AbsentCount   int `json:"absent_count"`
}

// SessionListItem represents a single session in the list
type SessionListItem struct {
	SessionID    *uuid.UUID `json:"session_id,omitempty"`
	WeekNumber   int16      `json:"week_number"`
	SessionType  string     `json:"session_type"`
	SessionDate  *string    `json:"session_date,omitempty"`
	PresentCount *int       `json:"present_count,omitempty"`
	AbsentCount  *int       `json:"absent_count,omitempty"`
	IsActive     *bool      `json:"is_active,omitempty"`
	Status       *string    `json:"status,omitempty"`
}

// GetCourseSessionsResponse is the response for course sessions list
type GetCourseSessionsResponse struct {
	CourseID     uuid.UUID         `json:"course_id"`
	CourseCode   string            `json:"course_code"`
	CourseName   string            `json:"course_name"`
	Semester     string            `json:"semester"`
	TotalWeeks   int16             `json:"total_weeks"`
	Sessions     []SessionListItem `json:"sessions"`
	OverallStats struct {
		CompletedSessions int `json:"completed_sessions"`
	} `json:"overall_stats"`
}

// StudentAttendanceStats represents attendance stats for a student
type StudentAttendanceStats struct {
	StudentID     uuid.UUID `json:"student_id"`
	StudentNumber string    `json:"student_number"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	PresentCount  int       `json:"present_count"`
}

// GetCourseStudentsResponse is the response for course students attendance
type GetCourseStudentsResponse struct {
	CourseID       uuid.UUID                `json:"course_id"`
	CourseCode     string                   `json:"course_code"`
	Semester       string                   `json:"semester"`
	TotalWeeks     int16                    `json:"total_weeks"`
	CompletedWeeks int                      `json:"completed_weeks"`
	Students       []StudentAttendanceStats `json:"students"`
}

// WeeklyAttendanceRecord represents attendance for a specific week
type WeeklyAttendanceRecord struct {
	Week        int16   `json:"week"`
	SessionType string  `json:"session_type"`
	Date        string  `json:"date"`
	MarkedVia   string  `json:"marked_via"`
	Note        *string `json:"note,omitempty"`
}

// CourseAttendanceDetail represents attendance detail for a course
type CourseAttendanceDetail struct {
	CourseID       uuid.UUID                `json:"course_id"`
	CourseCode     string                   `json:"course_code"`
	CourseName     string                   `json:"course_name"`
	Instructor     string                   `json:"instructor"`
	TotalWeeks     int16                    `json:"total_weeks"`
	Theory         *SessionTypeAttendance   `json:"theory,omitempty"`
	Lab            *SessionTypeAttendance   `json:"lab,omitempty"`
	WeeklyRecords  []WeeklyAttendanceRecord `json:"weekly_records"`
}

// GetMyAttendanceResponse is the response for student's own attendance
type GetMyAttendanceResponse struct {
	StudentID     uuid.UUID                `json:"student_id"`
	StudentNumber string                   `json:"student_number"`
	Semester      string                   `json:"semester"`
	Courses       []CourseAttendanceDetail `json:"courses"`
}

// SessionTypeAttendance represents attendance stats for a specific session type
type SessionTypeAttendance struct {
	PresentCount         int `json:"present_count"`
	AbsentCount          int `json:"absent_count"`
	TotalSessions        int `json:"total_sessions"`
	MinRequired          int `json:"min_required"`
	Passed               bool `json:"passed"`
}

// FailedStudent represents a student who failed due to attendance
type FailedStudent struct {
	StudentID     uuid.UUID              `json:"student_id"`
	StudentNumber string                 `json:"student_number"`
	StudentName   string                 `json:"student_name"`
	FailedType    string                 `json:"failed_type"` // "theory", "lab", or "both"
	Theory        *SessionTypeAttendance `json:"theory,omitempty"`
	Lab           *SessionTypeAttendance `json:"lab,omitempty"`
}

// FinalizeAttendanceResponse is the response for finalization
type FinalizeAttendanceResponse struct {
	CourseID      uuid.UUID `json:"course_id"`
	CourseCode    string    `json:"course_code"`
	Semester      string    `json:"semester"`
	TotalStudents int       `json:"total_students"`
	TotalWeeks    int16     `json:"total_weeks"`
	Thresholds    struct {
		TheoryMinRequired int `json:"theory_min_required"`
		LabMinRequired    int `json:"lab_min_required"`
	} `json:"thresholds"`
	FinalizationSummary struct {
		PassingCount int `json:"passing_count"`
		FailingCount int `json:"failing_count"`
	} `json:"finalization_summary"`
	FailedStudents  []FailedStudent `json:"failed_students"`
	EventsPublished int             `json:"events_published"`
	FinalizedAt     time.Time       `json:"finalized_at"`
}

// AdminSessionItem represents a session in admin list view
type AdminSessionItem struct {
	SessionID     uuid.UUID `json:"session_id"`
	CourseID      uuid.UUID `json:"course_id"`
	CourseCode    string    `json:"course_code"`
	CourseName    string    `json:"course_name"`
	InstructorID  uuid.UUID `json:"instructor_id"`
	Semester      string    `json:"semester"`
	WeekNumber    int16     `json:"week_number"`
	SessionType   string    `json:"session_type"`
	SessionDate   string    `json:"session_date"`
	IsActive      bool      `json:"is_active"`
	StartedAt     time.Time `json:"started_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	PresentCount  int64     `json:"present_count"`
	EnrolledCount int64     `json:"enrolled_count"`
}

// AdminSessionsResponse is the response for admin sessions list
type AdminSessionsResponse struct {
	Sessions []AdminSessionItem `json:"sessions"`
	Total    int                `json:"total"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// GetSessionDetailsResponse is the response for session details
type GetSessionDetailsResponse struct {
	SessionID            uuid.UUID `json:"session_id"`
	CourseID             uuid.UUID `json:"course_id"`
	CourseCode           string    `json:"course_code"`
	CourseName           string    `json:"course_name"`
	WeekNumber           int16     `json:"week_number"`
	SessionType          string    `json:"session_type"`
	SessionDate          string    `json:"session_date"`
	Semester             string    `json:"semester"`
	IsActive             bool      `json:"is_active"`
	StartedAt            time.Time `json:"started_at"`
	ExpiresAt            time.Time `json:"expires_at"`
	EnrolledStudentCount int       `json:"enrolled_student_count"`
	PresentCount         int       `json:"present_count"`
	AbsentCount          int       `json:"absent_count"`
}

// AttendanceRecordItem represents a single attendance record
type AttendanceRecordItem struct {
	ID            uuid.UUID  `json:"id"`
	StudentID     uuid.UUID  `json:"student_id"`
	StudentNumber string     `json:"student_number"`
	StudentName   string     `json:"student_name"`
	MarkedVia     string     `json:"marked_via"`
	MarkedAt      *time.Time `json:"marked_at,omitempty"`
	Note          *string    `json:"note,omitempty"`
}

// GetSessionRecordsResponse is the response for session attendance records
type GetSessionRecordsResponse struct {
	SessionID    uuid.UUID              `json:"session_id"`
	WeekNumber   int16                  `json:"week_number"`
	PresentCount int                    `json:"present_count"`
	Records      []AttendanceRecordItem `json:"records"`
}

// EnrolledStudentItem represents a student enrolled in the course
type EnrolledStudentItem struct {
	StudentID     uuid.UUID `json:"student_id"`
	StudentNumber string    `json:"student_number"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Email         string    `json:"email"`
	IsMarked      bool      `json:"is_marked"`
}

// GetSessionStudentsResponse is the response for enrolled students
type GetSessionStudentsResponse struct {
	SessionID     uuid.UUID             `json:"session_id"`
	CourseID      uuid.UUID             `json:"course_id"`
	TotalEnrolled int                   `json:"total_enrolled"`
	MarkedCount   int                   `json:"marked_count"`
	Students      []EnrolledStudentItem `json:"students"`
}
