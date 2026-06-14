package dto

import (
	"time"

	"github.com/google/uuid"
)

// AvailableCoursesRequest represents the request to get available courses
type AvailableCoursesRequest struct {
	Semester string `form:"semester" binding:"required"`
}

// AvailableCourse represents a course available for enrollment
type AvailableCourse struct {
	ID                uuid.UUID         `json:"id"`
	CourseCode        string            `json:"course_code"`
	CourseName        string            `json:"course_name"`
	Credits           int16             `json:"credits"`
	ScheduleSessions  []ScheduleSession `json:"schedule_sessions"`
	MaxCapacity       int16             `json:"max_capacity"`
	CurrentEnrollment int16             `json:"current_enrollment"`
	AvailableSeats    int16             `json:"available_seats"`
	Instructor        string            `json:"instructor"`
}

// AvailableCoursesResponse represents the response with available courses
type AvailableCoursesResponse struct {
	StudentID        uuid.UUID         `json:"student_id"`
	Department       string            `json:"department"`
	ClassLevel       int16             `json:"class_level"`
	Semester         string            `json:"semester"`
	AvailableCourses []AvailableCourse `json:"available_courses"`
}

// CreateEnrollmentRequest represents the request to create an enrollment program
type CreateEnrollmentRequest struct {
	StudentID uuid.UUID   `json:"student_id"` // Set from JWT in handler, not from request body
	Semester  string      `json:"semester" binding:"required"`
	CourseIDs []uuid.UUID `json:"course_ids" binding:"required,min=1"`
}

// EnrollmentProgramResponse represents an enrollment program with courses
type EnrollmentProgramResponse struct {
	ID            uuid.UUID     `json:"id"`
	StudentID     uuid.UUID     `json:"student_id"`
	StudentNumber string        `json:"student_number,omitempty"`
	StudentName   string        `json:"student_name,omitempty"`
	Department    string        `json:"department,omitempty"`
	ClassLevel    int16         `json:"class_level,omitempty"`
	Semester      string        `json:"semester"`
	Status        string        `json:"status"`
	Courses       []CourseBasic `json:"courses"`
	CreatedAt     time.Time     `json:"created_at"`
}

// MyEnrollmentsRequest represents the request to get student's enrollments
type MyEnrollmentsRequest struct {
	Semester string `form:"semester"`
	Status   string `form:"status"`
}

// MyEnrollmentsResponse represents the response with student's enrollment programs
type MyEnrollmentsResponse struct {
	StudentID uuid.UUID                   `json:"student_id"`
	Programs  []EnrollmentProgramResponse `json:"programs"`
}

// LatestRejectionRequest represents the request to get latest rejection
type LatestRejectionRequest struct {
	Semester string `form:"semester" binding:"required"`
}

// RejectedCourseDetail represents a rejected course detail
type RejectedCourseDetail struct {
	CourseID   uuid.UUID `json:"course_id"`
	CourseCode string    `json:"course_code"`
	CourseName string    `json:"course_name"`
	Credits    int16     `json:"credits"`
	Instructor string    `json:"instructor"`
}

// RejectedCoursesData represents the JSONB structure of rejected courses
type RejectedCoursesData struct {
	Courses      []RejectedCourseDetail `json:"courses"`
	TotalCredits int                    `json:"total_credits"`
	SubmittedAt  time.Time              `json:"submitted_at"`
}

// RejectionDetail represents a rejection log detail
type RejectionDetail struct {
	ID              uuid.UUID           `json:"id"`
	AdvisorID       uuid.UUID           `json:"advisor_id"`
	AdvisorFullname string              `json:"advisor_fullname"`
	RejectionReason string              `json:"rejection_reason"`
	RejectedCourses RejectedCoursesData `json:"rejected_courses"`
	RejectedAt      time.Time           `json:"rejected_at"`
}

// LatestRejectionResponse represents the response with latest rejection
type LatestRejectionResponse struct {
	StudentID       uuid.UUID        `json:"student_id"`
	Semester        string           `json:"semester"`
	HasRejection    bool             `json:"has_rejection"`
	LatestRejection *RejectionDetail `json:"latest_rejection"`
	TotalRejections int64            `json:"total_rejections"`
}

// MyRejectionsRequest represents the request to get all rejections
type MyRejectionsRequest struct {
	Semester string `form:"semester"`
	PaginationRequest
}

// MyRejectionsResponse represents the response with all rejections
type MyRejectionsResponse struct {
	StudentID  uuid.UUID          `json:"student_id"`
	Rejections []RejectionDetail  `json:"rejections"`
	Pagination PaginationResponse `json:"pagination"`
}

// ApproveEnrollmentRequest represents the request to approve an enrollment
type ApproveEnrollmentRequest struct {
	ProgramID uuid.UUID `json:"program_id" binding:"required"`
}

// RejectEnrollmentRequest represents the request to reject an enrollment
type RejectEnrollmentRequest struct {
	ProgramID       uuid.UUID `json:"program_id" binding:"required"`
	RejectionReason string    `json:"rejection_reason" binding:"required"`
}

// AdvisorPendingProgramsResponse represents pending programs for advisor review
type AdvisorPendingProgramsResponse struct {
	AdvisorID uuid.UUID                   `json:"advisor_id"`
	Programs  []EnrollmentProgramResponse `json:"programs"`
}
