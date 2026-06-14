package dto

import (
	"time"

	"github.com/google/uuid"
)

// Base event structure
type BaseEvent struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

// ========== Inbound Events (Consumed from RabbitMQ) ==========

// StudentCreatedEvent represents student.created event
type StudentCreatedEvent struct {
	BaseEvent
	StudentID     uuid.UUID `json:"student_id"`
	StudentNumber string    `json:"student_number"`
	Email         string    `json:"email"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Department    string    `json:"department"`
	ClassLevel    int16     `json:"class_level"`
	AdvisorID     uuid.UUID `json:"advisor_id"`
	Status        string    `json:"status"`
}

// StudentUpdatedEvent represents student.updated event
type StudentUpdatedEvent struct {
	BaseEvent
	StudentID     uuid.UUID  `json:"student_id"`
	StudentNumber string     `json:"student_number"`
	Email         string     `json:"email"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	Department    string     `json:"department"`
	ClassLevel    int16      `json:"class_level"`
	AdvisorID     *uuid.UUID `json:"advisor_id"`
	Status        string     `json:"status"`
}

// StudentDeactivatedEvent represents student.deactivated event
type StudentDeactivatedEvent struct {
	BaseEvent
	StudentID uuid.UUID `json:"student_id"`
}

// CourseSession represents a course schedule session
type CourseSession struct {
	DayOfWeek   string `json:"day_of_week"`
	SlotNumbers []int  `json:"slot_numbers"`
	SessionType string `json:"session_type"`
}

// PrerequisiteCourse represents a prerequisite course
type PrerequisiteCourse struct {
	ID         uuid.UUID `json:"id"`
	CourseCode string    `json:"course_code"`
	CourseName string    `json:"course_name"`
}

// CourseSemesterCreatedEvent represents course.semester.created event
type CourseSemesterCreatedEvent struct {
	BaseEvent
	SemesterCourseID  uuid.UUID            `json:"semester_course_id"`
	CourseCode        string               `json:"course_code"`
	CourseName        string               `json:"course_name"`
	Faculty           string               `json:"faculty"`
	Department        string               `json:"department"`
	Credits           int16                `json:"credits"`
	CourseType        string               `json:"course_type"`
	ClassLevel        int16                `json:"class_level"`
	Semester          string               `json:"semester"`
	InstructorID      *uuid.UUID           `json:"instructor_id"`
	InstructorFullname string              `json:"instructor_fullname"`
	ClassroomLocation string               `json:"classroom_location"`
	MaxCapacity       int16                `json:"max_capacity"`
	Prerequisites     []PrerequisiteCourse `json:"prerequisites"`
	ScheduleSessions  []CourseSession      `json:"schedule_sessions"`
}

// GradeStudentPrerequisitePassedEvent represents grade.student.prerequisite.passed event
type GradeStudentPrerequisitePassedEvent struct {
	BaseEvent
	StudentID  uuid.UUID `json:"student_id"`
	CourseCode string    `json:"course_code"`
	Semester   string    `json:"semester"`
	GradePoint string    `json:"grade_point"`
}

// ========== Outbound Events (Published to RabbitMQ via Outbox) ==========

// EnrollmentProgramSubmittedEvent represents enrollment.program_submitted event
type EnrollmentProgramSubmittedEvent struct {
	BaseEvent
	ProgramID  uuid.UUID   `json:"program_id"`
	StudentID  uuid.UUID   `json:"student_id"`
	Semester   string      `json:"semester"`
	CourseIDs  []uuid.UUID `json:"course_ids"`
	TotalCourses int       `json:"total_courses"`
}

// EnrollmentProgramApprovedEvent represents enrollment.program_approved event
type EnrollmentProgramApprovedEvent struct {
	BaseEvent
	ProgramID   uuid.UUID   `json:"program_id"`
	StudentID   uuid.UUID   `json:"student_id"`
	Semester    string      `json:"semester"`
	CourseIDs   []uuid.UUID `json:"course_ids"`
	ApprovedBy  uuid.UUID   `json:"approved_by"`
	ApprovedAt  time.Time   `json:"approved_at"`
}

// EnrollmentProgramRejectedEvent represents enrollment.program_rejected event
type EnrollmentProgramRejectedEvent struct {
	BaseEvent
	ProgramID       uuid.UUID   `json:"program_id"`
	StudentID       uuid.UUID   `json:"student_id"`
	Semester        string      `json:"semester"`
	CourseIDs       []uuid.UUID `json:"course_ids"`
	RejectedBy      uuid.UUID   `json:"rejected_by"`
	RejectionReason string      `json:"rejection_reason"`
	RejectedAt      time.Time   `json:"rejected_at"`
}
