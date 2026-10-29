package service

import (
	"context"
	"encoding/json"

	"github.com/baaaki/mydreamcampus/enrollment/internal/db"
	"github.com/baaaki/mydreamcampus/enrollment/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/enrollment/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/contracts"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ApproveEnrollmentProgram approves a student's enrollment program. The
// student's advisor decides; an admin may decide for any student, which is
// the only way a student without an advisor (none assigned yet, or the
// advisor deactivated) gets an answer.
func (s *EnrollmentService) ApproveEnrollmentProgram(ctx context.Context, programID uuid.UUID, callerID uuid.UUID, isAdmin bool) (dto.EnrollmentProgramResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "ApproveEnrollmentProgram"),
		zap.String("program_id", programID.String()),
		zap.String("caller_id", callerID.String()),
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
	if program.Status.EnrollmentStatusEnum == db.EnrollmentEnrollmentStatusEnumApproved {
		serviceLogger.Warn("program already approved")
		return dto.EnrollmentProgramResponse{}, serviceErrors.ErrCannotModifyApproved
	}

	// Get student to verify advisor
	student, err := lookupStudent(ctx, s.studentClient, utils.PgtypeToUUID(program.StudentID))
	if err != nil {
		return dto.EnrollmentProgramResponse{}, err
	}

	if !isAdmin && !isAdvisorOf(student, callerID) {
		serviceLogger.Warn("advisor mismatch")
		// 403, not 401: the caller is signed in. A 401 makes the web client
		// refresh and then log the user out.
		return dto.EnrollmentProgramResponse{}, sharedErrors.ErrForbidden
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
		ApprovedBy: callerID,
	})

	// Approve program (with event)
	approvedProgram, err := s.enrollmentRepo.ApproveProgramWithEvent(ctx, programID, eventPayload)
	if err != nil {
		if sharedErrors.Is(err, serviceErrors.ErrProgramNotPending) {
			serviceLogger.Warn("program no longer pending")
			return dto.EnrollmentProgramResponse{}, err
		}
		return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("enrollment program approved successfully")

	// The same course details my-enrollments shows, so the advisor's screen
	// can draw the approved program without another read.
	catalog := s.catalogCourses(ctx, approvedProgram.Semester, courseIDs)
	coursesDTO := make([]dto.CourseBasic, 0, len(coursesRows))
	for _, row := range coursesRows {
		course := catalog[utils.PgtypeToUUID(row.CourseID)]
		coursesDTO = append(coursesDTO, dto.CourseBasic{
			ID:               utils.PgtypeToUUID(row.CourseID).String(),
			CourseCode:       row.CourseCode,
			CourseName:       row.CourseName,
			Credits:          row.Credits,
			InstructorName:   course.InstructorFullname,
			ScheduleSessions: toScheduleSessions(course.ScheduleSessions),
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

// RejectEnrollmentProgram rejects a student's enrollment program. Who may
// decide is the same as for ApproveEnrollmentProgram.
func (s *EnrollmentService) RejectEnrollmentProgram(ctx context.Context, programID uuid.UUID, callerID uuid.UUID, isAdmin bool, rejectionReason string) error {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "RejectEnrollmentProgram"),
		zap.String("program_id", programID.String()),
		zap.String("caller_id", callerID.String()),
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
	if program.Status.EnrollmentStatusEnum == db.EnrollmentEnrollmentStatusEnumApproved {
		serviceLogger.Warn("cannot reject approved program")
		return serviceErrors.ErrCannotModifyApproved
	}

	// Get student to verify advisor
	student, err := lookupStudent(ctx, s.studentClient, utils.PgtypeToUUID(program.StudentID))
	if err != nil {
		return err
	}

	if !isAdmin && !isAdvisorOf(student, callerID) {
		serviceLogger.Warn("advisor mismatch")
		return sharedErrors.ErrForbidden
	}

	// The name the student reads on the rejection. The token carries no
	// names, the student's record carries the advisor's; an admin deciding
	// in a missing advisor's place is shown as such.
	rejectedBy := "Yönetici"
	if isAdvisorOf(student, callerID) && student.AdvisorName != nil {
		rejectedBy = *student.AdvisorName
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
	}
	catalog := s.catalogCourses(ctx, program.Semester, courseIDs)

	for _, row := range coursesRows {
		courseID := utils.PgtypeToUUID(row.CourseID)
		rejectedCourseDetails = append(rejectedCourseDetails, dto.RejectedCourseDetail{
			CourseID:   courseID,
			CourseCode: row.CourseCode,
			CourseName: row.CourseName,
			Credits:    row.Credits,
			Instructor: catalog[courseID].InstructorFullname,
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
		AdvisorID:         utils.UUIDToPgtype(callerID),
		AdvisorFullname:   rejectedBy,
		Semester:          program.Semester,
		RejectionReason:   rejectionReason,
		RejectedCourses:   rejectedCoursesJSON,
	}

	eventPayload := buildEnrollmentRejectedPayload(EnrollmentRejectedInputs{
		ProgramID:       programID,
		StudentID:       utils.PgtypeToUUID(program.StudentID),
		Semester:        program.Semester,
		CourseIDs:       courseIDs,
		RejectedBy:      callerID,
		RejectionReason: rejectionReason,
	})

	// Reject program
	err = s.enrollmentRepo.RejectProgramWithEventAndLog(ctx, programID, rejectionLogParams, eventPayload)
	if err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("enrollment program rejected successfully")

	return nil
}

// GetPendingProgramsByAdvisor returns the pending programs of an advisor's
// students. For an admin it returns those of the students without an
// advisor, which no advisor's list holds.
func (s *EnrollmentService) GetPendingProgramsByAdvisor(ctx context.Context, advisorID uuid.UUID, isAdmin bool) (dto.AdvisorPendingProgramsResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "GetPendingProgramsByAdvisor"),
		zap.String("advisor_id", advisorID.String()),
	)

	var students []contracts.StudentResponse
	var err error
	if isAdmin {
		students, err = s.studentClient.GetStudentsWithoutAdvisor(ctx)
	} else {
		students, err = s.studentClient.GetStudentsByAdvisorID(ctx, advisorID)
	}
	if err != nil {
		return dto.AdvisorPendingProgramsResponse{}, upstreamError(err)
	}

	if len(students) == 0 {
		return dto.AdvisorPendingProgramsResponse{
			AdvisorID: advisorID,
			Programs:  []dto.EnrollmentProgramResponse{},
		}, nil
	}

	studentIDs := make([]uuid.UUID, len(students))
	studentMap := make(map[uuid.UUID]contracts.StudentResponse, len(students))
	for i, st := range students {
		id, _ := uuid.Parse(st.ID)
		studentIDs[i] = id
		studentMap[id] = st
	}

	programs, err := s.enrollmentRepo.GetPendingProgramsByStudentIDs(ctx, studentIDs)
	if err != nil {
		return dto.AdvisorPendingProgramsResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	type pendingProgram struct {
		program db.EnrollmentProgram
		courses []db.EnrollmentProgramCourse
	}
	loaded := make([]pendingProgram, 0, len(programs))
	courseIDsBySemester := make(map[string][]uuid.UUID)
	seen := make(map[uuid.UUID]bool)
	for _, program := range programs {
		programID := utils.PgtypeToUUID(program.ID)
		coursesRows, err := s.enrollmentRepo.GetCoursesByProgramID(ctx, programID)
		if err != nil {
			serviceLogger.Error("failed to get courses for program",
				zap.String("program_id", programID.String()),
				zap.Error(err),
			)
			continue
		}
		loaded = append(loaded, pendingProgram{program: program, courses: coursesRows})
		for _, row := range coursesRows {
			courseID := utils.PgtypeToUUID(row.CourseID)
			if !seen[courseID] {
				seen[courseID] = true
				courseIDsBySemester[program.Semester] = append(courseIDsBySemester[program.Semester], courseID)
			}
		}
	}

	// Advisees share most of their courses, so each offering is asked for
	// once per semester rather than once per program. The instructor is only
	// a label on the review dialog: a catalog outage leaves it blank instead
	// of hiding the programs waiting for a decision.
	instructors := make(map[uuid.UUID]string, len(seen))
	for semester, ids := range courseIDsBySemester {
		courses, err := s.courseCatalogClient.GetCoursesByIDs(ctx, semester, ids)
		if err != nil {
			serviceLogger.Warn("failed to get instructors from catalog",
				zap.String("semester", semester),
				zap.Error(err),
			)
			continue
		}
		for _, c := range courses {
			instructors[c.ID] = c.InstructorFullname
		}
	}

	programsDTO := make([]dto.EnrollmentProgramResponse, 0, len(loaded))
	for _, p := range loaded {
		studentID := utils.PgtypeToUUID(p.program.StudentID)
		student := studentMap[studentID]

		coursesDTO := make([]dto.CourseBasic, 0, len(p.courses))
		for _, row := range p.courses {
			courseID := utils.PgtypeToUUID(row.CourseID)
			coursesDTO = append(coursesDTO, dto.CourseBasic{
				ID:               courseID.String(),
				CourseCode:       row.CourseCode,
				CourseName:       row.CourseName,
				Credits:          row.Credits,
				InstructorName:   instructors[courseID],
				ScheduleSessions: []dto.ScheduleSession{},
			})
		}

		programsDTO = append(programsDTO, dto.EnrollmentProgramResponse{
			ID:            utils.PgtypeToUUID(p.program.ID),
			StudentID:     studentID,
			StudentNumber: student.StudentNumber,
			StudentName:   student.FirstName + " " + student.LastName,
			Department:    student.Department,
			ClassLevel:    student.ClassLevel,
			Semester:      p.program.Semester,
			Status:        string(p.program.Status.EnrollmentStatusEnum),
			Courses:       coursesDTO,
			CreatedAt:     p.program.CreatedAt.Time,
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

func isAdvisorOf(student contracts.StudentResponse, callerID uuid.UUID) bool {
	return student.AdvisorID != nil && *student.AdvisorID == callerID.String()
}

// catalogCourses reads courses from the catalog for their instructor and
// weekly sessions. They only label a program, so a catalog outage leaves
// them out rather than failing a decision already made.
func (s *EnrollmentService) catalogCourses(ctx context.Context, semester string, ids []uuid.UUID) map[uuid.UUID]contracts.SemesterCourseResponse {
	out := make(map[uuid.UUID]contracts.SemesterCourseResponse, len(ids))
	courses, err := s.courseCatalogClient.GetCoursesByIDs(ctx, semester, ids)
	if err != nil {
		logger.WithContextAndFields(ctx, zap.String("method", "catalogCourses")).
			Warn("failed to read course details from catalog", zap.Error(err))
		return out
	}
	for _, c := range courses {
		out[c.ID] = c
	}
	return out
}
