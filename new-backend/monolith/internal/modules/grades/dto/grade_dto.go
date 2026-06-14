package dto

import (
	"time"

	"github.com/google/uuid"
)

// ============================================
// Score Submission DTOs
// ============================================

type SubmitScoreRequest struct {
	RegistrationID uuid.UUID `json:"registration_id" binding:"required"`
	Slug           string    `json:"slug" binding:"required"`
	Score          *float64  `json:"score"`
	IsAbsent       bool      `json:"is_absent"`
}

type BulkSubmitScoresRequest struct {
	Slug   string            `json:"slug" binding:"required"`
	Scores []BulkScoreEntry  `json:"scores" binding:"required,dive"`
}

type BulkScoreEntry struct {
	RegistrationID uuid.UUID `json:"registration_id" binding:"required"`
	Score          *float64  `json:"score"`
	IsAbsent       bool      `json:"is_absent"`
}

type SubmitScoreResponse struct {
	ID            uuid.UUID        `json:"id"`
	StudentNumber string           `json:"student_number"`
	Slug          string           `json:"slug"`
	Score         *float64         `json:"score"`
	IsAbsent      bool             `json:"is_absent"`
	GradedAt      time.Time        `json:"graded_at"`
	AutoFinalized bool             `json:"auto_finalized,omitempty"`
	FinalizeResult *FinalizeResult `json:"finalize_result,omitempty"`
}

type BulkSubmitScoresResponse struct {
	Slug           string          `json:"slug"`
	SuccessCount   int             `json:"success_count"`
	AutoFinalized  bool            `json:"auto_finalized,omitempty"`
	FinalizeResult *FinalizeResult `json:"finalize_result,omitempty"`
}

type FinalizeResult struct {
	GradingType           string  `json:"grading_type"`
	ClassMean             float64 `json:"class_mean"`
	TotalStudents         int     `json:"total_students"`
	PassingCount          int     `json:"passing_count"`
	FailingCount          int     `json:"failing_count"`
	AttendanceFailedCount int     `json:"attendance_failed_count,omitempty"`
}

// ============================================
// Course Status DTOs
// ============================================

type CourseStatusResponse struct {
	CourseID       uuid.UUID            `json:"course_id"`
	CourseCode     string               `json:"course_code"`
	CourseName     string               `json:"course_name"`
	Semester       string               `json:"semester"`
	TotalStudents  int                  `json:"total_students"`
	Assessments    []AssessmentStatus   `json:"assessments"`
	IsFinalized    bool                 `json:"is_finalized"`
	PendingMessage string               `json:"pending_message,omitempty"`
	FinalizedAt    *time.Time           `json:"finalized_at,omitempty"`
	GradingType    string               `json:"grading_type,omitempty"`
	ClassStats     *ClassStatistics     `json:"class_statistics,omitempty"`
}

type AssessmentStatus struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Weight       int    `json:"weight"`
	GradedCount  int    `json:"graded_count"`
	PendingCount int    `json:"pending_count"`
	IsComplete   bool   `json:"is_complete"`
}

type ClassStatistics struct {
	Mean                  float64 `json:"mean"`
	StdDev                float64 `json:"stddev"`
	PassingCount          int     `json:"passing_count"`
	FailingCount          int     `json:"failing_count"`
	AttendanceFailedCount int     `json:"attendance_failed_count,omitempty"`
}

// ============================================
// Course Students DTOs
// ============================================

type CourseStudentsResponse struct {
	CourseID   uuid.UUID       `json:"course_id"`
	CourseCode string          `json:"course_code"`
	Students   []StudentGrades `json:"students"`
}

type StudentGrades struct {
	RegistrationID     uuid.UUID              `json:"registration_id"`
	StudentID          uuid.UUID              `json:"student_id"`
	StudentNumber      string                 `json:"student_number"`
	FirstName          string                 `json:"first_name"`
	LastName           string                 `json:"last_name"`
	Scores             map[string]ScoreDetail `json:"scores"`
	CurrentAverage     *float64               `json:"current_average"`
	IsAttendanceFailed bool                   `json:"is_attendance_failed,omitempty"`
}

type ScoreDetail struct {
	Score    *float64 `json:"score"`
	IsAbsent bool     `json:"is_absent"`
	IsLocked bool     `json:"is_locked"`
}

// ============================================
// Student My Grades DTOs
// ============================================

type MyGradesResponse struct {
	StudentID        uuid.UUID           `json:"student_id"`
	StudentNumber    string              `json:"student_number"`
	ActiveCourses    []ActiveCourse      `json:"active_courses"`
	CompletedCourses []CompletedCourse   `json:"completed_courses"`
	CumulativeGPA    float64             `json:"cumulative_gpa"`
	TotalCredits     int                 `json:"total_credits"`
}

type ActiveCourse struct {
	CourseCode string                 `json:"course_code"`
	CourseName string                 `json:"course_name"`
	Semester   string                 `json:"semester"`
	Credits    int                    `json:"credits"`
	Scores     map[string]ScoreDetail `json:"scores"`
}

type CompletedCourse struct {
	CourseCode       string             `json:"course_code"`
	CourseName       string             `json:"course_name"`
	Semester         string             `json:"semester"`
	Credits          int                `json:"credits"`
	WeightedAverage  float64            `json:"weighted_average"`
	GradePoint       string             `json:"grade_point"`
	AssessmentScores map[string]float64 `json:"assessment_scores,omitempty"`
}

// ============================================
// Transcript DTOs
// ============================================

type TranscriptResponse struct {
	Student  StudentInfo       `json:"student"`
	Semesters []SemesterGrades `json:"semesters"`
	Summary  TranscriptSummary `json:"summary"`
	GeneratedAt time.Time      `json:"generated_at"`
}

type StudentInfo struct {
	StudentNumber  string `json:"student_number"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Department     string `json:"department"`
	EnrollmentYear int    `json:"enrollment_year"`
}

type SemesterGrades struct {
	Semester        string          `json:"semester"`
	SemesterDisplay string          `json:"semester_display"`
	Courses         []CourseGrade   `json:"courses"`
	SemesterCredits int             `json:"semester_credits"`
	SemesterGPA     float64         `json:"semester_gpa"`
}

type CourseGrade struct {
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	Credits    int    `json:"credits"`
	GradePoint string `json:"grade_point"`
}

type TranscriptSummary struct {
	TotalCredits  int     `json:"total_credits"`
	CumulativeGPA float64 `json:"cumulative_gpa"`
}

// ============================================
// Score Lock/Unlock DTOs
// ============================================

type ScoreLockRequest struct {
	RegistrationID uuid.UUID `json:"registration_id" binding:"required"`
	Slug           string    `json:"slug" binding:"required"`
}

// LockAssessmentResponse is the result of an instructor locking an entire
// assessment (all students for a given slug). When every slug of the course
// is locked, the course auto-finalizes and FinalizeTriggered is true.
type LockAssessmentResponse struct {
	CourseID          uuid.UUID `json:"course_id"`
	Slug              string    `json:"slug"`
	LockedCount       int       `json:"locked_count"`
	FinalizeTriggered bool      `json:"finalize_triggered,omitempty"`
}

// ============================================
// Appeal (İtiraz) DTOs
// ============================================

type AppealScoreRequest struct {
	StudentID uuid.UUID `json:"student_id" binding:"required"`
	CourseID  uuid.UUID `json:"course_id" binding:"required"`
	Slug      string    `json:"slug" binding:"required"`
	NewScore  float64   `json:"new_score" binding:"required,min=0,max=100"`
	Reason    string    `json:"reason" binding:"required,min=10"`
}

type AppealScoreResponse struct {
	StudentID          uuid.UUID `json:"student_id"`
	CourseCode         string    `json:"course_code"`
	Slug               string    `json:"slug"`
	OldScore           float64   `json:"old_score"`
	NewScore           float64   `json:"new_score"`
	OldWeightedAverage float64   `json:"old_weighted_average"`
	NewWeightedAverage float64   `json:"new_weighted_average"`
	OldGradePoint      string    `json:"old_grade_point"`
	NewGradePoint      string    `json:"new_grade_point"`
	GradingType        string    `json:"grading_type"`
	FrozenClassMean    float64   `json:"frozen_class_mean"`
	FrozenClassStdDev  float64   `json:"frozen_class_stddev,omitempty"`
}
