package service

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/enrollment-service/internal/db"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/dto"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GetMyEnrollments returns student's enrollment programs
func (s *EnrollmentService) GetMyEnrollments(ctx context.Context, studentID uuid.UUID, semester *string, status *string) (dto.MyEnrollmentsResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "GetMyEnrollments"),
		zap.String("student_id", studentID.String()),
	)

	// Get programs
	programs, err := s.enrollmentRepo.GetEnrollmentProgramsByStudent(ctx, studentID, semester, status)
	if err != nil {
		return dto.MyEnrollmentsResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Build response with course details
	programsDTO := make([]dto.EnrollmentProgramResponse, 0, len(programs))
	for _, program := range programs {
		programID := utils.PgtypeToUUID(program.ID)

		// Get courses for this program
		coursesRows, err := s.enrollmentRepo.GetCoursesByProgramID(ctx, programID)
		if err != nil {
			serviceLogger.Error("failed to get courses for program",
				zap.String("program_id", programID.String()),
				zap.Error(err),
			)
			continue
		}

		// Collect course IDs for session lookup
		allCourseIDs := make([]uuid.UUID, len(coursesRows))
		for i, row := range coursesRows {
			allCourseIDs[i] = utils.PgtypeToUUID(row.CourseID)
		}

		// Get sessions for these courses
		sessions, err := s.courseRepo.GetSessionsByCourseIDs(ctx, allCourseIDs)
		if err != nil {
			serviceLogger.Warn("failed to get sessions for program courses",
				zap.String("program_id", programID.String()),
				zap.Error(err),
			)
			// Continue without sessions
		}

		// Group sessions by course_id
		sessionsMap := make(map[uuid.UUID][]db.CourseSessionsCache)
		for _, session := range sessions {
			cID := utils.PgtypeToUUID(session.CourseID)
			sessionsMap[cID] = append(sessionsMap[cID], session)
		}

		coursesDTO := make([]dto.CourseBasic, 0, len(coursesRows))
		for _, row := range coursesRows {
			cID := utils.PgtypeToUUID(row.CourseID)

			// Map sessions
			var scheduleSessions []dto.ScheduleSession
			if sessList, ok := sessionsMap[cID]; ok {
				daySessionsMap := make(map[string][]int)
				for _, s := range sessList {
					day := string(s.DayOfWeek)
					daySessionsMap[day] = append(daySessionsMap[day], int(s.SlotNumber))
				}
				for day, slots := range daySessionsMap {
					scheduleSessions = append(scheduleSessions, dto.ScheduleSession{
						DayOfWeek:   day,
						SlotNumbers: slots,
					})
				}
			}

			coursesDTO = append(coursesDTO, dto.CourseBasic{
				ID:               cID.String(),
				CourseCode:       row.CourseCode,
				CourseName:       utils.PgTextToString(row.CourseName),
				Credits:          row.Credits,
				InstructorName:   utils.PgTextToString(row.InstructorFullname),
				ScheduleSessions: scheduleSessions,
			})
		}

		programsDTO = append(programsDTO, dto.EnrollmentProgramResponse{
			ID:        programID,
			StudentID: utils.PgtypeToUUID(program.StudentID),
			Semester:  program.Semester,
			Status:    string(program.Status.EnrollmentStatusEnum),
			Courses:   coursesDTO,
			CreatedAt: program.CreatedAt.Time,
		})
	}

	serviceLogger.Info("enrollment programs retrieved",
		zap.Int("program_count", len(programsDTO)),
	)

	return dto.MyEnrollmentsResponse{
		StudentID: studentID,
		Programs:  programsDTO,
	}, nil
}

// GetLatestRejection returns student's latest rejection for a semester
func (s *EnrollmentService) GetLatestRejection(ctx context.Context, studentID uuid.UUID, semester string) (dto.LatestRejectionResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "GetLatestRejection"),
		zap.String("student_id", studentID.String()),
		zap.String("semester", semester),
	)

	// Get latest rejection
	rejection, err := s.enrollmentRepo.GetLatestRejectionByStudentAndSemester(ctx, studentID, semester)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrNotFound) {
			// No rejection found
			count, _ := s.enrollmentRepo.CountRejectionsByStudentAndSemester(ctx, studentID, semester)
			return dto.LatestRejectionResponse{
				StudentID:       studentID,
				Semester:        semester,
				HasRejection:    false,
				LatestRejection: nil,
				TotalRejections: count,
			}, nil
		}
		return dto.LatestRejectionResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Parse rejected courses JSONB
	var rejectedCoursesData dto.RejectedCoursesData
	if err := json.Unmarshal(rejection.RejectedCourses, &rejectedCoursesData); err != nil {
		serviceLogger.Error("failed to parse rejected courses JSON",
			zap.Error(err),
		)
		return dto.LatestRejectionResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Count total rejections
	count, err := s.enrollmentRepo.CountRejectionsByStudentAndSemester(ctx, studentID, semester)
	if err != nil {
		return dto.LatestRejectionResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("latest rejection retrieved",
		zap.Int64("total_rejections", count),
	)

	return dto.LatestRejectionResponse{
		StudentID:    studentID,
		Semester:     semester,
		HasRejection: true,
		LatestRejection: &dto.RejectionDetail{
			ID:              utils.PgtypeToUUID(rejection.ID),
			AdvisorID:       utils.PgtypeToUUID(rejection.AdvisorID),
			AdvisorFullname: rejection.AdvisorFullname,
			RejectionReason: rejection.RejectionReason,
			RejectedCourses: rejectedCoursesData,
			RejectedAt:      rejection.RejectedAt.Time,
		},
		TotalRejections: count,
	}, nil
}

// GetMyRejections returns all rejections for a student
func (s *EnrollmentService) GetMyRejections(ctx context.Context, studentID uuid.UUID, semester *string) (dto.MyRejectionsResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "GetMyRejections"),
		zap.String("student_id", studentID.String()),
	)

	// Get rejections
	rejections, err := s.enrollmentRepo.GetRejectionsByStudentAndSemester(ctx, studentID, semester)
	if err != nil {
		return dto.MyRejectionsResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Build response
	rejectionsDTO := make([]dto.RejectionDetail, 0, len(rejections))
	for _, rejection := range rejections {
		var rejectedCoursesData dto.RejectedCoursesData
		if err := json.Unmarshal(rejection.RejectedCourses, &rejectedCoursesData); err != nil {
			serviceLogger.Error("failed to parse rejected courses JSON",
				zap.String("rejection_id", utils.PgtypeToUUID(rejection.ID).String()),
				zap.Error(err),
			)
			continue
		}

		rejectionsDTO = append(rejectionsDTO, dto.RejectionDetail{
			ID:              utils.PgtypeToUUID(rejection.ID),
			AdvisorID:       utils.PgtypeToUUID(rejection.AdvisorID),
			AdvisorFullname: rejection.AdvisorFullname,
			RejectionReason: rejection.RejectionReason,
			RejectedCourses: rejectedCoursesData,
			RejectedAt:      rejection.RejectedAt.Time,
		})
	}

	serviceLogger.Info("rejections retrieved",
		zap.Int("rejection_count", len(rejectionsDTO)),
	)

	return dto.MyRejectionsResponse{
		StudentID:  studentID,
		Rejections: rejectionsDTO,
		Pagination: dto.PaginationResponse{
			Page:       1,
			Limit:      len(rejectionsDTO),
			Total:      len(rejectionsDTO),
			TotalPages: 1,
		},
	}, nil
}
