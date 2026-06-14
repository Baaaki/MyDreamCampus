package service

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/enrollment-service/internal/db"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/enrollment-service/internal/errors"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ApproveEnrollmentProgram approves a student's enrollment program
func (s *EnrollmentService) ApproveEnrollmentProgram(ctx context.Context, programID uuid.UUID, advisorID uuid.UUID) (dto.EnrollmentProgramResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "ApproveEnrollmentProgram"),
		zap.String("program_id", programID.String()),
		zap.String("advisor_id", advisorID.String()),
	)

	// Get program
	program, err := s.enrollmentRepo.GetEnrollmentProgramByID(ctx, programID)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrNotFound) {
			serviceLogger.Warn("program not found")
			return dto.EnrollmentProgramResponse{}, serviceErrors.ErrProgramNotFound
		}
		return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Check if already approved
	if program.Status.EnrollmentStatusEnum == db.EnrollmentStatusEnumApproved {
		serviceLogger.Warn("program already approved")
		return dto.EnrollmentProgramResponse{}, serviceErrors.ErrCannotModifyApproved
	}

	// Get student to verify advisor
	student, err := s.studentRepo.GetStudentByID(ctx, utils.PgtypeToUUID(program.StudentID))
	if err != nil {
		return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Verify advisor — student must have an assigned advisor and it must match
	if !student.AdvisorID.Valid || utils.PgtypeToUUID(student.AdvisorID) != advisorID {
		serviceLogger.Warn("advisor mismatch",
			zap.String("requesting_advisor", advisorID.String()),
		)
		return dto.EnrollmentProgramResponse{}, sharedErrors.ErrUnauthorized
	}

	// Get courses
	coursesRows, err := s.enrollmentRepo.GetCoursesByProgramID(ctx, programID)
	if err != nil {
		return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	courseIDs := make([]uuid.UUID, len(coursesRows))
	for i, row := range coursesRows {
		courseIDs[i] = utils.PgtypeToUUID(row.CourseID)
	}

	eventPayload := buildEnrollmentApprovedPayload(EnrollmentApprovedInputs{
		ProgramID:  programID,
		StudentID:  utils.PgtypeToUUID(program.StudentID),
		Semester:   program.Semester,
		CourseIDs:  courseIDs,
		ApprovedBy: advisorID,
	})

	// Approve program (with event)
	approvedProgram, err := s.enrollmentRepo.ApproveProgramWithEvent(ctx, programID, eventPayload)
	if err != nil {
		return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("enrollment program approved successfully")

	// Build response
	coursesDTO := make([]dto.CourseBasic, 0, len(coursesRows))
	for _, row := range coursesRows {
		coursesDTO = append(coursesDTO, dto.CourseBasic{
			ID:             utils.PgtypeToUUID(row.CourseID).String(),
			CourseCode:     row.CourseCode,
			CourseName:     utils.PgTextToString(row.CourseName),
			Credits:        row.Credits,
			InstructorName: utils.PgTextToString(row.InstructorFullname),
		})
	}

	return dto.EnrollmentProgramResponse{
		ID:        utils.PgtypeToUUID(approvedProgram.ID),
		StudentID: utils.PgtypeToUUID(approvedProgram.StudentID),
		Semester:  approvedProgram.Semester,
		Status:    string(approvedProgram.Status.EnrollmentStatusEnum),
		Courses:   coursesDTO,
		CreatedAt: approvedProgram.CreatedAt.Time,
	}, nil
}

// RejectEnrollmentProgram rejects a student's enrollment program
func (s *EnrollmentService) RejectEnrollmentProgram(ctx context.Context, programID uuid.UUID, advisorID uuid.UUID, advisorFullname string, rejectionReason string) error {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "RejectEnrollmentProgram"),
		zap.String("program_id", programID.String()),
		zap.String("advisor_id", advisorID.String()),
	)

	// Get program
	program, err := s.enrollmentRepo.GetEnrollmentProgramByID(ctx, programID)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrNotFound) {
			serviceLogger.Warn("program not found")
			return serviceErrors.ErrProgramNotFound
		}
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Check if already approved
	if program.Status.EnrollmentStatusEnum == db.EnrollmentStatusEnumApproved {
		serviceLogger.Warn("cannot reject approved program")
		return serviceErrors.ErrCannotModifyApproved
	}

	// Get student to verify advisor
	student, err := s.studentRepo.GetStudentByID(ctx, utils.PgtypeToUUID(program.StudentID))
	if err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Verify advisor — student must have an assigned advisor and it must match
	if !student.AdvisorID.Valid || utils.PgtypeToUUID(student.AdvisorID) != advisorID {
		serviceLogger.Warn("advisor mismatch",
			zap.String("requesting_advisor", advisorID.String()),
		)
		return sharedErrors.ErrUnauthorized
	}

	// Get courses
	coursesRows, err := s.enrollmentRepo.GetCoursesByProgramID(ctx, programID)
	if err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Build rejected courses snapshot
	rejectedCourseDetails := make([]dto.RejectedCourseDetail, 0, len(coursesRows))
	totalCredits := 0
	courseIDs := make([]uuid.UUID, len(coursesRows))

	for i, row := range coursesRows {
		courseIDs[i] = utils.PgtypeToUUID(row.CourseID)
		rejectedCourseDetails = append(rejectedCourseDetails, dto.RejectedCourseDetail{
			CourseID:   utils.PgtypeToUUID(row.CourseID),
			CourseCode: row.CourseCode,
			CourseName: utils.PgTextToString(row.CourseName),
			Credits:    row.Credits,
			Instructor: utils.PgTextToString(row.InstructorFullname),
		})
		totalCredits += int(row.Credits)
	}

	rejectedCoursesData := dto.RejectedCoursesData{
		Courses:      rejectedCourseDetails,
		TotalCredits: totalCredits,
		SubmittedAt:  program.CreatedAt.Time,
	}

	rejectedCoursesJSON, err := json.Marshal(rejectedCoursesData)
	if err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Create rejection log params
	rejectionLogParams := db.CreateRejectionLogParams{
		OriginalProgramID: utils.UUIDToPgtype(programID),
		StudentID:         program.StudentID,
		AdvisorID:         utils.UUIDToPgtype(advisorID),
		AdvisorFullname:   advisorFullname,
		Semester:          program.Semester,
		RejectionReason:   rejectionReason,
		RejectedCourses:   rejectedCoursesJSON,
	}

	eventPayload := buildEnrollmentRejectedPayload(EnrollmentRejectedInputs{
		ProgramID:       programID,
		StudentID:       utils.PgtypeToUUID(program.StudentID),
		Semester:        program.Semester,
		CourseIDs:       courseIDs,
		RejectedBy:      advisorID,
		RejectionReason: rejectionReason,
	})

	// Reject program (create log, decrement enrollments, delete program, create event)
	err = s.enrollmentRepo.RejectProgramWithEventAndLog(ctx, programID, rejectionLogParams, courseIDs, eventPayload)
	if err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("enrollment program rejected successfully")

	return nil
}

// GetPendingProgramsByAdvisor returns pending programs for an advisor
func (s *EnrollmentService) GetPendingProgramsByAdvisor(ctx context.Context, advisorID uuid.UUID) (dto.AdvisorPendingProgramsResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "GetPendingProgramsByAdvisor"),
		zap.String("advisor_id", advisorID.String()),
	)

	// Get pending programs
	programs, err := s.enrollmentRepo.GetPendingProgramsByAdvisor(ctx, advisorID)
	if err != nil {
		return dto.AdvisorPendingProgramsResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
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

		coursesDTO := make([]dto.CourseBasic, 0, len(coursesRows))
		for _, row := range coursesRows {
			coursesDTO = append(coursesDTO, dto.CourseBasic{
				ID:             utils.PgtypeToUUID(row.CourseID).String(),
				CourseCode:     row.CourseCode,
				CourseName:     utils.PgTextToString(row.CourseName),
				Credits:        row.Credits,
				InstructorName: utils.PgTextToString(row.InstructorFullname),
			})
		}

		programsDTO = append(programsDTO, dto.EnrollmentProgramResponse{
			ID:            programID,
			StudentID:     utils.PgtypeToUUID(program.StudentID),
			StudentName:   utils.PgTextToString(program.FirstName) + " " + utils.PgTextToString(program.LastName),
			StudentNumber: program.StudentNumber,
			Department:    utils.PgTextToString(program.Department),
			ClassLevel:    int16(program.ClassLevel.Int16),
			Semester:      program.Semester,
			Status:        string(program.Status.EnrollmentStatusEnum),
			Courses:       coursesDTO,
			CreatedAt:     program.CreatedAt.Time,
		})
	}

	serviceLogger.Info("pending programs retrieved for advisor",
		zap.Int("program_count", len(programsDTO)),
	)

	return dto.AdvisorPendingProgramsResponse{
		AdvisorID: advisorID,
		Programs:  programsDTO,
	}, nil
}
