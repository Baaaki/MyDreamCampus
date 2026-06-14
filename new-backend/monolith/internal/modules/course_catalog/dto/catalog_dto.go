package dto

import (
	"time"

	"github.com/google/uuid"
)

// Prerequisite represents a prerequisite course
type Prerequisite struct {
	ID         uuid.UUID `json:"id"`
	CourseCode string    `json:"course_code"`
	CourseName string    `json:"course_name"`
}

// CourseCoordinator represents the course coordinator information
type CourseCoordinator struct {
	Title  string `json:"title"`
	Name   string `json:"name"`
	Email  string `json:"email,omitempty"`
	Phone  string `json:"phone,omitempty"`
	Office string `json:"office,omitempty"`
}

// WeeklyTopic represents a weekly topic in the course syllabus
type WeeklyTopic struct {
	Week        int    `json:"week"`
	Topic       string `json:"topic"`
	Description string `json:"description,omitempty"`
}

// CreateCourseRequest represents the request to create a new course
type CreateCourseRequest struct {
	CourseCode   string  `json:"course_code" binding:"required,min=2,max=50"`
	Name         string  `json:"name" binding:"required,min=3,max=255"`
	Faculty      string  `json:"faculty" binding:"required,min=3,max=100"`
	Department   string  `json:"department" binding:"required,min=3,max=100"`
	OfferingUnit *string `json:"offering_unit" binding:"omitempty,max=255"`

	// Dönem ve seviye bilgileri
	ClassLevel int16  `json:"class_level" binding:"required,min=1,max=6"`
	Semester   *int16 `json:"semester" binding:"omitempty,min=1,max=8"`

	// Kredi ve saat bilgileri
	Credits          int16  `json:"credits" binding:"min=0,max=30"`
	ECTS             *int16 `json:"ects" binding:"omitempty,min=0,max=60"`
	TheoreticalHours int16  `json:"theoretical_hours" binding:"omitempty,min=0,max=20"`
	LabHours         int16  `json:"lab_hours" binding:"omitempty,min=0,max=20"`

	// Ders tipleri ve kategorileri
	CourseType     string  `json:"course_type" binding:"required,oneof=mandatory elective"`
	CourseCategory string  `json:"course_category" binding:"omitempty,oneof=theoretical practical internship project seminar"`
	EducationLevel string  `json:"education_level" binding:"omitempty,oneof=undergraduate graduate doctorate"`
	TeachingType   string  `json:"teaching_type" binding:"omitempty,oneof=on_campus online hybrid"`
	Language       *string `json:"language" binding:"omitempty,max=50"`

	// İlişkisel veriler
	Prerequisites []Prerequisite     `json:"prerequisites" binding:"omitempty,dive"`
	Coordinator   *CourseCoordinator `json:"coordinator" binding:"omitempty"`

	// Ders içeriği
	Purpose              *string       `json:"purpose" binding:"omitempty,max=5000"`
	Description          *string       `json:"description" binding:"omitempty,max=5000"`
	LearningOutcomes     *string       `json:"learning_outcomes" binding:"omitempty,max=5000"`
	LearningOutcomesList []string      `json:"learning_outcomes_list" binding:"omitempty,dive,max=500"`
	WeeklyTopics         []WeeklyTopic `json:"weekly_topics" binding:"omitempty,dive"`
	RecommendedSources   []string      `json:"recommended_sources" binding:"omitempty,dive,max=500"`
	Syllabus             *string       `json:"syllabus" binding:"omitempty,max=10000"`

	Status string `json:"status" binding:"omitempty,oneof=active draft pending_approval under_revision archived suspended"`
}

// UpdateCourseRequest represents the request to update a course
type UpdateCourseRequest struct {
	Name         *string `json:"name" binding:"omitempty,min=3,max=255"`
	Faculty      *string `json:"faculty" binding:"omitempty,min=3,max=100"`
	Department   *string `json:"department" binding:"omitempty,min=3,max=100"`
	OfferingUnit *string `json:"offering_unit" binding:"omitempty,max=255"`

	// Dönem ve seviye bilgileri
	ClassLevel *int16 `json:"class_level" binding:"omitempty,min=1,max=6"`
	Semester   *int16 `json:"semester" binding:"omitempty,min=1,max=8"`

	// Kredi ve saat bilgileri
	Credits          *int16 `json:"credits" binding:"omitempty,min=0,max=30"`
	ECTS             *int16 `json:"ects" binding:"omitempty,min=0,max=60"`
	TheoreticalHours *int16 `json:"theoretical_hours" binding:"omitempty,min=0,max=20"`
	LabHours         *int16 `json:"lab_hours" binding:"omitempty,min=0,max=20"`

	// Ders tipleri ve kategorileri
	CourseType     *string `json:"course_type" binding:"omitempty,oneof=mandatory elective"`
	CourseCategory *string `json:"course_category" binding:"omitempty,oneof=theoretical practical internship project seminar"`
	EducationLevel *string `json:"education_level" binding:"omitempty,oneof=undergraduate graduate doctorate"`
	TeachingType   *string `json:"teaching_type" binding:"omitempty,oneof=on_campus online hybrid"`
	Language       *string `json:"language" binding:"omitempty,max=50"`

	// İlişkisel veriler
	Prerequisites *[]Prerequisite    `json:"prerequisites" binding:"omitempty,dive"`
	Coordinator   *CourseCoordinator `json:"coordinator" binding:"omitempty"`

	// Ders içeriği
	Purpose              *string        `json:"purpose" binding:"omitempty,max=5000"`
	Description          *string        `json:"description" binding:"omitempty,max=5000"`
	LearningOutcomes     *string        `json:"learning_outcomes" binding:"omitempty,max=5000"`
	LearningOutcomesList *[]string      `json:"learning_outcomes_list" binding:"omitempty,dive,max=500"`
	WeeklyTopics         *[]WeeklyTopic `json:"weekly_topics" binding:"omitempty,dive"`
	RecommendedSources   *[]string      `json:"recommended_sources" binding:"omitempty,dive,max=500"`
	Syllabus             *string        `json:"syllabus" binding:"omitempty,max=10000"`

	Status *string `json:"status" binding:"omitempty,oneof=active draft pending_approval under_revision archived suspended"`
}

// CourseResponse represents a course in API responses
type CourseResponse struct {
	ID           uuid.UUID `json:"id"`
	CourseCode   string    `json:"course_code"`
	Name         string    `json:"name"`
	Faculty      string    `json:"faculty"`
	Department   string    `json:"department"`
	OfferingUnit *string   `json:"offering_unit,omitempty"`

	// Dönem ve seviye bilgileri
	ClassLevel int16  `json:"class_level"`
	Semester   *int16 `json:"semester,omitempty"`

	// Kredi ve saat bilgileri
	Credits          int16  `json:"credits"`
	ECTS             *int16 `json:"ects,omitempty"`
	TheoreticalHours int16  `json:"theoretical_hours"`
	LabHours         int16  `json:"lab_hours"`

	// Ders tipleri ve kategorileri
	CourseType     string `json:"course_type"`
	CourseCategory string `json:"course_category"`
	EducationLevel string `json:"education_level"`
	TeachingType   string `json:"teaching_type"`
	Language       string `json:"language"`

	// İlişkisel veriler
	Prerequisites []Prerequisite     `json:"prerequisites"`
	Coordinator   *CourseCoordinator `json:"coordinator,omitempty"`

	// Ders içeriği
	Purpose              *string       `json:"purpose,omitempty"`
	Description          *string       `json:"description,omitempty"`
	LearningOutcomes     *string       `json:"learning_outcomes,omitempty"`
	LearningOutcomesList []string      `json:"learning_outcomes_list,omitempty"`
	WeeklyTopics         []WeeklyTopic `json:"weekly_topics,omitempty"`
	RecommendedSources   []string      `json:"recommended_sources,omitempty"`
	Syllabus             *string       `json:"syllabus,omitempty"`

	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CourseListItem represents a course in list responses (minimal fields)
type CourseListItem struct {
	ID           uuid.UUID `json:"id"`
	CourseCode   string    `json:"course_code"`
	Name         string    `json:"name"`
	Faculty      string    `json:"faculty"`
	Department   string    `json:"department"`
	OfferingUnit *string   `json:"offering_unit,omitempty"`

	// Dönem ve seviye bilgileri
	ClassLevel int16  `json:"class_level"`
	Semester   *int16 `json:"semester,omitempty"`

	// Kredi ve saat bilgileri
	Credits          int16  `json:"credits"`
	ECTS             *int16 `json:"ects,omitempty"`
	TheoreticalHours int16  `json:"theoretical_hours"`
	LabHours         int16  `json:"lab_hours"`

	// Ders tipleri ve kategorileri
	CourseType     string `json:"course_type"`
	CourseCategory string `json:"course_category"`
	EducationLevel string `json:"education_level"`
	TeachingType   string `json:"teaching_type"`
	Language       string `json:"language"`

	Prerequisites []Prerequisite `json:"prerequisites"`
	Status        string         `json:"status"`
}

// ListCoursesRequest represents query parameters for listing courses
type ListCoursesRequest struct {
	PaginationRequest
	Faculty        *string `form:"faculty" binding:"omitempty"`
	Department     *string `form:"department" binding:"omitempty"`
	CourseType     *string `form:"course_type" binding:"omitempty,oneof=mandatory elective"`
	CourseCategory *string `form:"course_category" binding:"omitempty,oneof=theoretical practical internship project seminar"`
	EducationLevel *string `form:"education_level" binding:"omitempty,oneof=undergraduate graduate doctorate"`
	Status         *string `form:"status" binding:"omitempty,oneof=active draft pending_approval under_revision archived suspended"`
	ClassLevel     *int16  `form:"class_level" binding:"omitempty,min=1,max=6"`
	Semester       *int16  `form:"semester" binding:"omitempty,min=1,max=8"`
	Language       *string `form:"language" binding:"omitempty,max=50"`
	Search         *string `form:"search" binding:"omitempty,max=100"`
}

// ListCoursesResponse represents the response for listing courses
type ListCoursesResponse struct {
	Data       []CourseListItem   `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}
