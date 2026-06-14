package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/dto"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type StudentGradesService struct {
	cacheRepo        *repository.CacheRepository
	registrationRepo *repository.RegistrationRepository
	scoreRepo        *repository.ScoreRepository
	completedRepo    *repository.CompletedRepository
}

func NewStudentGradesService(
	cacheRepo *repository.CacheRepository,
	registrationRepo *repository.RegistrationRepository,
	scoreRepo *repository.ScoreRepository,
	completedRepo *repository.CompletedRepository,
) *StudentGradesService {
	return &StudentGradesService{
		cacheRepo:        cacheRepo,
		registrationRepo: registrationRepo,
		scoreRepo:        scoreRepo,
		completedRepo:    completedRepo,
	}
}

func (s *StudentGradesService) GetMyGrades(ctx context.Context, studentID uuid.UUID) (*dto.MyGradesResponse, error) {
	log := logger.WithContextAndFields(ctx,
		zap.String("service", "StudentGradesService"),
		zap.String("method", "GetMyGrades"),
	)

	// 1. Check if student is active
	student, err := s.cacheRepo.GetStudentCacheByID(ctx, studentID)
	if err != nil {
		log.Error("failed to get student", zap.Error(err))
		return nil, errors.ErrStudentNotFound
	}

	if !utils.PgBoolToBool(student.IsActive) {
		return nil, errors.ErrStudentDeactivated
	}

	// 2. Get active courses (registrations not yet finalized into completed_courses)
	activeRows, err := s.registrationRepo.GetActiveRegistrationsByStudent(ctx, studentID)
	if err != nil {
		log.Error("failed to get active registrations", zap.Error(err))
		return nil, err
	}

	activeCourses := make([]dto.ActiveCourse, 0, len(activeRows))
	for _, ar := range activeRows {
		scoresMap, err := decodeScoresJSON(ar.Scores)
		if err != nil {
			log.Error("failed to decode active course scores",
				zap.String("course_code", ar.CourseCode),
				zap.Error(err))
			scoresMap = map[string]dto.ScoreDetail{}
		}
		activeCourses = append(activeCourses, dto.ActiveCourse{
			CourseCode: ar.CourseCode,
			CourseName: ar.CourseName,
			Semester:   ar.Semester,
			Credits:    int(ar.Credits),
			Scores:     scoresMap,
		})
	}

	// 3. Get completed courses
	completedCourses, err := s.completedRepo.GetCompletedCoursesByStudent(ctx, studentID)
	if err != nil {
		log.Error("failed to get completed courses", zap.Error(err))
		return nil, err
	}

	var completedCoursesDTO []dto.CompletedCourse
	for _, cc := range completedCourses {
		weightedAvg, err := utils.PgNumericToFloat64(cc.WeightedAverage)
		if err != nil {
			log.Error("failed to convert weighted average", zap.Error(err))
			weightedAvg = 0.0
		}

		// Parse assessment scores from JSONB
		var assessmentScores map[string]float64
		if len(cc.AssessmentScores) > 0 {
			if err := json.Unmarshal(cc.AssessmentScores, &assessmentScores); err != nil {
				log.Error("failed to unmarshal assessment scores", zap.Error(err))
				assessmentScores = nil
			}
		}

		completedCoursesDTO = append(completedCoursesDTO, dto.CompletedCourse{
			CourseCode:       cc.CourseCode,
			CourseName:       cc.CourseName,
			Semester:         cc.Semester,
			Credits:          int(cc.Credits),
			WeightedAverage:  weightedAvg,
			GradePoint:       string(cc.GradePoint),
			AssessmentScores: assessmentScores,
		})
	}

	// 4. Calculate GPA
	gpaResult, err := s.completedRepo.CalculateStudentGPA(ctx, studentID)
	if err != nil {
		log.Error("failed to calculate GPA", zap.Error(err))
		return nil, err
	}

	gpa := parseInterfaceToFloat64(gpaResult.Gpa)
	totalCredits := parseInterfaceToInt64(gpaResult.TotalCredits)

	return &dto.MyGradesResponse{
		StudentID:        studentID,
		StudentNumber:    student.StudentNumber,
		ActiveCourses:    activeCourses,
		CompletedCourses: completedCoursesDTO,
		CumulativeGPA:    gpa,
		TotalCredits:     int(totalCredits),
	}, nil
}

func (s *StudentGradesService) GetTranscript(ctx context.Context, requesterID uuid.UUID, requesterRole string, studentID uuid.UUID) (*dto.TranscriptResponse, error) {
	// 1. Authorization check
	if requesterRole == "student" && requesterID != studentID {
		return nil, errors.ErrNotCourseInstructor // Reusing error for unauthorized access
	}

	// 2. Check if student is active (only for student role)
	student, err := s.cacheRepo.GetStudentCacheByID(ctx, studentID)
	if err != nil {
		logger.Error("failed to get student", zap.Error(err))
		return nil, errors.ErrStudentNotFound
	}

	if requesterRole == "student" && !utils.PgBoolToBool(student.IsActive) {
		return nil, errors.ErrStudentDeactivated
	}

	// 3. Get transcript data
	transcriptData, err := s.completedRepo.GetTranscriptData(ctx, studentID)
	if err != nil {
		logger.Error("failed to get transcript data", zap.Error(err))
		return nil, err
	}

	// 4. Group by semester
	semesterMap := make(map[string]*dto.SemesterGrades)
	for _, td := range transcriptData {
		semester := td.Semester

		if _, exists := semesterMap[semester]; !exists {
			semesterMap[semester] = &dto.SemesterGrades{
				Semester:        semester,
				SemesterDisplay: formatSemesterDisplay(semester),
				Courses:         []dto.CourseGrade{},
			}
		}

		semesterMap[semester].Courses = append(semesterMap[semester].Courses, dto.CourseGrade{
			CourseCode: td.CourseCode,
			CourseName: td.CourseName,
			Credits:    int(td.Credits),
			GradePoint: string(td.GradePoint),
		})
	}

	// 5. Calculate semester GPAs and credits
	var semesters []dto.SemesterGrades
	for _, sem := range semesterMap {
		semesterCredits := 0
		semesterWeightedSum := 0.0

		for _, course := range sem.Courses {
			semesterCredits += course.Credits
			gradeValue := gradePointToFloat(course.GradePoint)
			semesterWeightedSum += gradeValue * float64(course.Credits)
		}

		sem.SemesterCredits = semesterCredits
		if semesterCredits > 0 {
			sem.SemesterGPA = semesterWeightedSum / float64(semesterCredits)
		}

		semesters = append(semesters, *sem)
	}

	// 6. Calculate overall GPA
	gpaResult, err := s.completedRepo.CalculateStudentGPA(ctx, studentID)
	if err != nil {
		logger.Error("failed to calculate GPA", zap.Error(err))
		return nil, err
	}

	gpa := parseInterfaceToFloat64(gpaResult.Gpa)
	totalCredits := parseInterfaceToInt64(gpaResult.TotalCredits)

	// 7. Extract enrollment year from student number (assuming format: 2021123456)
	enrollmentYear := 0
	if len(student.StudentNumber) >= 4 {
		if year, err := strconv.Atoi(student.StudentNumber[:4]); err == nil {
			enrollmentYear = year
		}
	}

	return &dto.TranscriptResponse{
		Student: dto.StudentInfo{
			StudentNumber:  student.StudentNumber,
			FirstName:      student.FirstName.String,
			LastName:       student.LastName.String,
			Department:     student.Department.String,
			EnrollmentYear: enrollmentYear,
		},
		Semesters: semesters,
		Summary: dto.TranscriptSummary{
			TotalCredits:  int(totalCredits),
			CumulativeGPA: gpa,
		},
	}, nil
}

// Helper functions

func formatSemesterDisplay(semester string) string {
	// Convert "2024_fall" to "2024-2025 Güz"
	parts := strings.Split(semester, "_")
	if len(parts) != 2 {
		return semester
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return semester
	}

	season := parts[1]
	seasonTR := map[string]string{
		"fall":   "Güz",
		"spring": "Bahar",
		"summer": "Yaz",
	}

	seasonName := seasonTR[season]
	if seasonName == "" {
		seasonName = season
	}

	if season == "fall" {
		return strconv.Itoa(year) + "-" + strconv.Itoa(year+1) + " " + seasonName
	}

	return strconv.Itoa(year) + " " + seasonName
}

func gradePointToFloat(gp string) float64 {
	// Convert "4.00" to 4.0
	val, err := strconv.ParseFloat(gp, 64)
	if err != nil {
		return 0.0
	}
	return val
}

// parseInterfaceToFloat64 safely converts interface{} (from pgx scan) to float64.
// PostgreSQL numeric/decimal types may arrive as pgtype.Numeric, string, or float64.
func parseInterfaceToFloat64(v any) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case pgtype.Numeric:
		f, err := val.Float64Value()
		if err != nil {
			return 0
		}
		return f.Float64
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		s := fmt.Sprintf("%v", val)
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
}

// decodeScoresJSON parses a JSONB blob (scanned as []byte) of the shape
//   { "midterm": { "score": 85, "is_absent": false, "is_locked": true }, ... }
// into the DTO map. Empty/nil input returns an empty (non-nil) map.
func decodeScoresJSON(raw any) (map[string]dto.ScoreDetail, error) {
	out := map[string]dto.ScoreDetail{}
	if raw == nil {
		return out, nil
	}
	var b []byte
	switch v := raw.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	case map[string]any:
		// pgx/v5 decodes jsonb into a Go map by default — re-marshal so the
		// downstream json.Unmarshal path stays single.
		marshaled, err := json.Marshal(v)
		if err != nil {
			return out, err
		}
		b = marshaled
	default:
		return out, fmt.Errorf("unexpected scores type %T", raw)
	}
	if len(b) == 0 || string(b) == "null" {
		return out, nil
	}

	// Backend stores numeric scores as DECIMAL — pgx renders them as JSON strings
	// inside jsonb_object_agg output ("85.00"). Decode into a flexible shape.
	type rawDetail struct {
		Score    json.Number `json:"score"`
		IsAbsent bool        `json:"is_absent"`
		IsLocked bool        `json:"is_locked"`
	}
	tmp := map[string]rawDetail{}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return out, err
	}
	for slug, d := range tmp {
		detail := dto.ScoreDetail{
			IsAbsent: d.IsAbsent,
			IsLocked: d.IsLocked,
		}
		if s := d.Score.String(); s != "" {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				detail.Score = &f
			}
		}
		out[slug] = detail
	}
	return out, nil
}

// parseInterfaceToInt64 safely converts interface{} (from pgx scan) to int64.
func parseInterfaceToInt64(v any) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case pgtype.Numeric:
		f, err := val.Float64Value()
		if err != nil {
			return 0
		}
		return int64(f.Float64)
	case int64:
		return val
	case int32:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	default:
		s := fmt.Sprintf("%v", val)
		i, _ := strconv.ParseInt(s, 10, 64)
		return i
	}
}
