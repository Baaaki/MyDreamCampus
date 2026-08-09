package dto

import (
	"github.com/baaaki/mydreamcampus/shared/contracts"
	"github.com/google/uuid"
)

// The course shapes below leave this service — enrollment reads them off
// /internal/semester-courses. Their canonical definition is in
// shared/contracts so both sides compile against one struct; the aliases
// keep the module's own code reading dto.X.
type (
	ScheduleSession        = contracts.ScheduleSession
	AssessmentItem         = contracts.AssessmentItem
	SemesterCourseResponse = contracts.SemesterCourseResponse
	SemesterCourseListItem = contracts.SemesterCourseListItem
)

// CreateSemesterCourseRequest represents the request to create a semester course
type CreateSemesterCourseRequest struct {
	CourseCode         string            `json:"course_code" binding:"required,min=2,max=50"`
	ClassLevel         int16             `json:"class_level" binding:"required,min=1,max=6"`
	InstructorID       uuid.UUID         `json:"instructor_id" binding:"required,uuid"`
	InstructorFullname string            `json:"instructor_fullname" binding:"required,min=3,max=150"`
	ClassroomLocation  string            `json:"classroom_location" binding:"required,min=3,max=100"`
	MaxCapacity        int16             `json:"max_capacity" binding:"required,min=1,max=1000"`
	AssessmentSchema   []AssessmentItem  `json:"assessment_schema" binding:"required,min=1,dive"`
	ScheduleSessions   []ScheduleSession `json:"schedule_sessions" binding:"required,min=1,dive"`
}

// ListSemesterCoursesRequest represents query parameters for listing semester courses
type ListSemesterCoursesRequest struct {
	PaginationRequest
	Faculty      *string    `form:"faculty" binding:"omitempty"`
	Department   *string    `form:"department" binding:"omitempty"`
	InstructorID *uuid.UUID `form:"instructor_id" binding:"omitempty,uuid"`
	CourseType   *string    `form:"course_type" binding:"omitempty,oneof=mandatory elective"`
	ClassLevel   *int16     `form:"class_level" binding:"omitempty,min=1,max=6"`
}

// ListSemesterCoursesResponse represents the response for listing semester courses
type ListSemesterCoursesResponse struct {
	Data       []SemesterCourseListItem `json:"data"`
	Pagination PaginationResponse       `json:"pagination"`
}

// DeleteSemesterCourseResponse represents the response for deleting a semester course
type DeleteSemesterCourseResponse struct {
	Message          string `json:"message"`
	SemesterCourseID string `json:"semester_course_id"`
	CourseCode       string `json:"course_code"`
	Semester         string `json:"semester"`
}

// TeacherScheduleSession represents a schedule session for teacher's course
type TeacherScheduleSession struct {
	Day         string `json:"day"`
	Time        string `json:"time"`
	Room        string `json:"room"`
	SessionType string `json:"session_type"` // "theory" or "lab"
}

// TeacherCourseItem represents a course item for teacher
type TeacherCourseItem struct {
	ID                uuid.UUID                `json:"id"`
	CourseCode        string                   `json:"course_code"`
	CourseName        string                   `json:"course_name"`
	Faculty           string                   `json:"faculty"`
	Department        string                   `json:"department"`
	Semester          string                   `json:"semester"`
	Credits           int16                    `json:"credits"`
	TheoreticalHours  int16                    `json:"theoretical_hours"`
	LabHours          int16                    `json:"lab_hours"`
	ClassroomLocation string                   `json:"classroom_location"`
	MaxCapacity       int16                    `json:"max_capacity"`
	Schedule          []TeacherScheduleSession `json:"schedule"`
}

// TeacherCoursesResponse represents the response for teacher's courses
type TeacherCoursesResponse struct {
	InstructorID uuid.UUID           `json:"instructor_id"`
	TotalCourses int                 `json:"total_courses"`
	Courses      []TeacherCourseItem `json:"courses"`
}
