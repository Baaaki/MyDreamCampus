package service

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/dto"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
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

		// Get courses from Course Catalog
		coursesFromCatalog, err := s.courseCatalogClient.GetCoursesByIDs(ctx, program.Semester, allCourseIDs)
		if err != nil {
			serviceLogger.Warn("failed to get courses from catalog",
				zap.String("program_id", programID.String()),
				zap.Error(err),
			)
			// Continue without full details if catalog fails, but we need session info.
		}

		// Create a map for quick lookup
		catalogMap := make(map[uuid.UUID]interface{})
		for _, c := range coursesFromCatalog {
			catalogMap[c.ID] = c.ScheduleSessions
		}

		coursesDTO := make([]dto.CourseBasic, 0, len(coursesRows))
		for _, row := range coursesRows {
			cID := utils.PgtypeToUUID(row.CourseID)

			var scheduleSessions []dto.ScheduleSession
			if val, ok := catalogMap[cID]; ok {
				if catalogSessions, ok := val.([]catalogDTO.ScheduleSession); ok {
					for _, s := range catalogSessions {
						var intSlots []int
						for _, sl := range s.SlotNumbers {
							intSlots = append(intSlots, int(sl))
						}
						scheduleSessions = append(scheduleSessions, dto.ScheduleSession{
							DayOfWeek:   s.DayOfWeek,
							SlotNumbers: intSlots,
							SessionType: s.SessionType,
						})
					}
				}
			}

			coursesDTO = append(coursesDTO, dto.CourseBasic{
				ID:               cID.String(),
				CourseCode:       row.CourseCode,
				CourseName:       row.CourseName,
				Credits:          row.Credits,
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
