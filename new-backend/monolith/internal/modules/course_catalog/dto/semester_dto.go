package dto

import (
	"time"

	"github.com/google/uuid"
)

// ScheduleSession represents a single schedule session (day + slot numbers + session type)
type ScheduleSession struct {
	DayOfWeek   string  `json:"day_of_week"`
	SlotNumbers []int16 `json:"slot_numbers"`
	SessionType string  `json:"session_type" binding:"required,oneof=theory lab"` // "theory" or "lab"
}

// AssessmentItem represents a single assessment component
type AssessmentItem struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Weight int16  `json:"weight"`
}

// CreateSemesterCourseRequest represents the request to create a semester course
type CreateSemesterCourseRequest struct {
	CourseCode         string             `json:"course_code" binding:"required,min=2,max=50"`
	ClassLevel         int16              `json:"class_level" binding:"required,min=1,max=6"`
	InstructorID       uuid.UUID          `json:"instructor_id" binding:"required,uuid"`
	InstructorFullname string             `json:"instructor_fullname" binding:"required,min=3,max=150"`
	ClassroomLocation  string             `json:"classroom_location" binding:"required,min=3,max=100"`
	MaxCapacity        int16              `json:"max_capacity" binding:"required,min=1,max=1000"`
	AssessmentSchema   []AssessmentItem   `json:"assessment_schema" binding:"required,min=1,dive"`
	ScheduleSessions   []ScheduleSession  `json:"schedule_sessions" binding:"required,min=1,dive"`
}

// SemesterCourseResponse represents a semester course in API responses
type SemesterCourseResponse struct {
	ID                 uuid.UUID         `json:"id"`
	Semester           string            `json:"semester"`
	CourseCode         string            `json:"course_code"`
	CourseName         string            `json:"course_name"`
	Department         string            `json:"department"`
	Credits            int16             `json:"credits"`
	ClassLevel         int16             `json:"class_level"`
	InstructorID       uuid.UUID         `json:"instructor_id"`
	InstructorFullname string            `json:"instructor_fullname"`
	ClassroomLocation  string            `json:"classroom_location"`
	MaxCapacity        int16             `json:"max_capacity"`
	AssessmentSchema   []AssessmentItem  `json:"assessment_schema"`
	ScheduleSessions   []ScheduleSession `json:"schedule_sessions"`
	Prerequisites      []Prerequisite    `json:"prerequisites,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// SemesterCourseListItem represents a semester course in list responses
type SemesterCourseListItem struct {
	ID                 uuid.UUID         `json:"id"`
	Semester           string            `json:"semester"`
	CourseCode         string            `json:"course_code"`
	CourseName         string            `json:"course_name"`
	Department         string            `json:"department"`
	Credits            int16             `json:"credits"`
	ClassLevel         int16             `json:"class_level"`
	InstructorID       uuid.UUID         `json:"instructor_id"`
	InstructorFullname string            `json:"instructor_fullname"`
	ClassroomLocation  string            `json:"classroom_location"`
	MaxCapacity        int16             `json:"max_capacity"`
	AssessmentSchema   []AssessmentItem  `json:"assessment_schema"`
	ScheduleSessions   []ScheduleSession `json:"schedule_sessions"`
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
