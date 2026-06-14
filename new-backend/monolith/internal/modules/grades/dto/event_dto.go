package dto

import (
	"time"

	"github.com/google/uuid"
)

// ============================================
// Inbound Events (Consumed from RabbitMQ)
// ============================================

// Student Events (nested structure - matches student-service event payload)
type StudentCreatedEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		ID            uuid.UUID `json:"id"`
		StudentNumber string    `json:"student_number"`
		FirstName     string    `json:"first_name"`
		LastName      string    `json:"last_name"`
		Email         string    `json:"email"`
		Department    string    `json:"department"`
		ClassLevel    int16     `json:"class_level"`
	} `json:"data"`
}

type StudentUpdatedEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		ID            uuid.UUID `json:"id"`
		StudentNumber string    `json:"student_number"`
		FirstName     string    `json:"first_name"`
		LastName      string    `json:"last_name"`
		Email         string    `json:"email"`
		Department    string    `json:"department"`
		ClassLevel    int16     `json:"class_level"`
	} `json:"data"`
}

type StudentDeactivatedEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		ID uuid.UUID `json:"id"`
	} `json:"data"`
}

// Course Events (flat structure - matches course-catalog-service event payload)
type CourseSemesterCreatedEvent struct {
	EventID            string                 `json:"event_id"`
	EventType          string                 `json:"event_type"`
	Timestamp          string                 `json:"timestamp"`
	SemesterCourseID   uuid.UUID              `json:"semester_course_id"`
	CourseCode         string                 `json:"course_code"`
	CourseName         string                 `json:"course_name"`
	Credits            int16                  `json:"credits"`
	Semester           string                 `json:"semester"`
	Department         string                 `json:"department"`
	InstructorID       uuid.UUID              `json:"instructor_id"`
	InstructorFullname string                 `json:"instructor_fullname"`
	AssessmentSchema   []AssessmentSchemaItem `json:"assessment_schema"`
}

type AssessmentSchemaItem struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

// Enrollment Events
type EnrollmentProgramApprovedEvent struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		StudentID  uuid.UUID   `json:"student_id"`
		Semester   string      `json:"semester"`
		CourseIDs  []uuid.UUID `json:"course_ids"`
		ApprovedAt time.Time   `json:"approved_at"`
	} `json:"data"`
}

// Attendance Events
type AttendanceSemesterFailedEvent struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		StudentID uuid.UUID `json:"student_id"`
		CourseID  uuid.UUID `json:"course_id"`
		Semester  string    `json:"semester"`
		FailedAt  time.Time `json:"failed_at"`
	} `json:"data"`
}

// ============================================
// Outbound Events (Published to RabbitMQ)
// ============================================

type GradeSubmittedEvent struct {
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		StudentID  uuid.UUID `json:"student_id"`
		CourseCode string    `json:"course_code"`
		Slug       string    `json:"slug"`
		Score      float64   `json:"score"`
	} `json:"data"`
}

// GradeFinalizeRequestedEvent is published by the grades-service back to itself
// when the last assessment of a course is locked. The finalize consumer picks
// this up and runs the heavy AutoFinalize computation off the request path.
type GradeFinalizeRequestedEvent struct {
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		CourseID     uuid.UUID `json:"course_id"`
		InstructorID uuid.UUID `json:"instructor_id"`
		TriggeredBy  string    `json:"triggered_by"`
	} `json:"data"`
}

type GradeFinalizedEvent struct {
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		CourseID              uuid.UUID `json:"course_id"`
		CourseCode            string    `json:"course_code"`
		Semester              string    `json:"semester"`
		GradingType           string    `json:"grading_type"`
		TotalStudents         int       `json:"total_students"`
		PassingCount          int       `json:"passing_count"`
		FailingCount          int       `json:"failing_count"`
		AttendanceFailedCount int       `json:"attendance_failed_count"`
		ClassMean             float64   `json:"class_mean"`
	} `json:"data"`
}

type GradeStudentPrerequisitePassedEvent struct {
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		StudentID  uuid.UUID `json:"student_id"`
		CourseID   uuid.UUID `json:"course_id"`
		CourseCode string    `json:"course_code"`
		Semester   string    `json:"semester"`
		GradePoint string    `json:"grade_point"`
	} `json:"data"`
}
