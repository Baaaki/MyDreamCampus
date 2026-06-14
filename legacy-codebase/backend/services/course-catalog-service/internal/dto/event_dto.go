package dto

import (
	"time"

	"github.com/google/uuid"
)

// CourseEvent represents the base structure for course events
type CourseEvent struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

// SemesterCourseCreatedEvent represents course.semester.created event
type SemesterCourseCreatedEvent struct {
	CourseEvent
	Data SemesterCourseEventData `json:"data"`
}

// SemesterCourseUpdatedEvent represents course.semester.updated event
type SemesterCourseUpdatedEvent struct {
	CourseEvent
	Data SemesterCourseEventData `json:"data"`
}

// SemesterCourseDeletedEvent represents course.semester.deleted event
type SemesterCourseDeletedEvent struct {
	CourseEvent
	Data SemesterCourseDeletedData `json:"data"`
}

// SemesterCourseEventData represents the data payload for semester course events
type SemesterCourseEventData struct {
	SemesterCourseID   uuid.UUID         `json:"semester_course_id"`
	Semester           string            `json:"semester"`
	CourseCode         string            `json:"course_code"`
	CourseName         string            `json:"course_name"`
	Faculty            string            `json:"faculty"`
	Department         string            `json:"department"`
	Credits            int16             `json:"credits"`
	ClassLevel         int16             `json:"class_level"`
	CourseType         string            `json:"course_type"`
	InstructorID       uuid.UUID         `json:"instructor_id"`
	InstructorFullname string            `json:"instructor_fullname"`
	ClassroomLocation  string            `json:"classroom_location"`
	MaxCapacity        int16             `json:"max_capacity"`
	AssessmentSchema   []AssessmentItem  `json:"assessment_schema"`
	Prerequisites      []Prerequisite    `json:"prerequisites"`
	ScheduleSessions   []ScheduleSession `json:"schedule_sessions"`
}

// SemesterCourseDeletedData represents the minimal data for delete events
type SemesterCourseDeletedData struct {
	SemesterCourseID uuid.UUID `json:"semester_course_id"`
	Semester         string    `json:"semester"`
	CourseCode       string    `json:"course_code"`
	CourseName       string    `json:"course_name"`
	Department       string    `json:"department"`
}

// InstructorChangedEvent represents course.instructor.changed event
type InstructorChangedEvent struct {
	CourseEvent
	Data InstructorChangedData `json:"data"`
}

// InstructorChangedData represents instructor change event data
type InstructorChangedData struct {
	SemesterCourseID       uuid.UUID `json:"semester_course_id"`
	Semester               string    `json:"semester"`
	CourseCode             string    `json:"course_code"`
	CourseName             string    `json:"course_name"`
	OldInstructorID        uuid.UUID `json:"old_instructor_id"`
	OldInstructorFullname  string    `json:"old_instructor_fullname"`
	NewInstructorID        uuid.UUID `json:"new_instructor_id"`
	NewInstructorFullname  string    `json:"new_instructor_fullname"`
}
