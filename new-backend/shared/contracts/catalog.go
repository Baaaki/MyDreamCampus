// Package contracts holds the types that cross a service boundary. A type
// belongs here only when a provider serves it and a consumer reads it — a
// module's internal DTOs stay in the module. Adding a field is backwards
// compatible; removing or renaming one breaks every consumer at once.
package contracts

import (
	"time"

	"github.com/google/uuid"
)

// AssessmentItem is a single graded component of a course.
type AssessmentItem struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Weight int16  `json:"weight"`
}

// ScheduleSession is one weekly meeting of a course (day + slots + type).
type ScheduleSession struct {
	DayOfWeek   string  `json:"day_of_week"`
	SlotNumbers []int16 `json:"slot_numbers"`
	SessionType string  `json:"session_type" binding:"required,oneof=theory lab"` // "theory" or "lab"
}

// Prerequisite is a course that must be passed before another one.
type Prerequisite struct {
	ID         uuid.UUID `json:"id"`
	CourseCode string    `json:"course_code"`
	CourseName string    `json:"course_name"`
}

// SemesterCourseListItem is the list projection of a semester course.
// Served by catalog on GET /internal/semester-courses, read by enrollment.
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

// SemesterCourseResponse is the full projection of a semester course.
// Served by catalog on GET /internal/semester-courses/:id, read by enrollment.
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

// SemesterInfo carries the deadline data attendance and grades enforce on.
// Served by catalog on GET /internal/semesters/:name/info.
type SemesterInfo struct {
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	HardDeadline   time.Time `json:"hard_deadline"`
	IsPastDeadline bool      `json:"is_past_deadline"`
}
