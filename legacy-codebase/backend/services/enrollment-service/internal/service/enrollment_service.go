package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/enrollment-service/internal/db"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/enrollment-service/internal/errors"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/repository"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/repository"
	"github.com/baaaki/mydreamcampus/shared/rules"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EnrollmentService struct {
	enrollmentRepo *repository.EnrollmentRepository
	studentRepo    *repository.StudentRepository
	courseRepo     *repository.CourseRepository
	periodRepo     *sharedRepo.SimplePeriodRepository
}

func NewEnrollmentService(
	enrollmentRepo *repository.EnrollmentRepository,
	studentRepo *repository.StudentRepository,
	courseRepo *repository.CourseRepository,
	periodRepo *sharedRepo.SimplePeriodRepository,
) *EnrollmentService {
	return &EnrollmentService{
		enrollmentRepo: enrollmentRepo,
		studentRepo:    studentRepo,
		courseRepo:     courseRepo,
		periodRepo:     periodRepo,
	}
}

// GetAvailableCourses returns courses available for a student
func (s *EnrollmentService) GetAvailableCourses(ctx context.Context, studentID uuid.UUID, semester string) (dto.AvailableCoursesResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "GetAvailableCourses"),
		zap.String("student_id", studentID.String()),
		zap.String("semester", semester),
	)

	// Get student from cache
	student, err := s.studentRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrNotFound) {
			serviceLogger.Warn("student not found in cache")
			return dto.AvailableCoursesResponse{}, serviceErrors.ErrStudentNotFound
		}
		return dto.AvailableCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Check if student is active
	if !student.IsActive.Bool || !student.IsActive.Valid {
		serviceLogger.Warn("student is deactivated")
		return dto.AvailableCoursesResponse{}, serviceErrors.ErrStudentDeactivated
	}

	// Get available courses
	courses, err := s.courseRepo.GetAvailableCourses(ctx,
		utils.PgTextToString(student.Department),
		student.ClassLevel.Int16,
		semester,
	)
	if err != nil {
		return dto.AvailableCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Get course IDs for session lookup
	courseIDs := make([]uuid.UUID, len(courses))
	for i, course := range courses {
		courseIDs[i] = utils.PgtypeToUUID(course.ID)
	}

	// Get sessions for all courses
	sessions, err := s.courseRepo.GetSessionsByCourseIDs(ctx, courseIDs)
	if err != nil {
		return dto.AvailableCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Group sessions by course_id
	sessionsMap := make(map[uuid.UUID][]db.CourseSessionsCache)
	for _, session := range sessions {
		courseID := utils.PgtypeToUUID(session.CourseID)
		sessionsMap[courseID] = append(sessionsMap[courseID], session)
	}

	// Build response
	availableCourses := make([]dto.AvailableCourse, 0, len(courses))
	for _, course := range courses {
		courseID := utils.PgtypeToUUID(course.ID)

		// Group sessions by day
		daySessionsMap := make(map[string][]int)
		if courseSessions, ok := sessionsMap[courseID]; ok {
			for _, session := range courseSessions {
				day := string(session.DayOfWeek)
				daySessionsMap[day] = append(daySessionsMap[day], int(session.SlotNumber))
			}
		}

		// Convert to DTO format
		scheduleSessions := make([]dto.ScheduleSession, 0, len(daySessionsMap))
		for day, slots := range daySessionsMap {
			scheduleSessions = append(scheduleSessions, dto.ScheduleSession{
				DayOfWeek:   day,
				SlotNumbers: slots,
			})
		}

		availableCourses = append(availableCourses, dto.AvailableCourse{
			ID:                courseID,
			CourseCode:        course.CourseCode,
			CourseName:        utils.PgTextToString(course.CourseName),
			Credits:           course.Credits,
			ScheduleSessions:  scheduleSessions,
			MaxCapacity:       course.MaxCapacity,
			CurrentEnrollment: course.CurrentEnrollment.Int16,
			AvailableSeats:    course.MaxCapacity - course.CurrentEnrollment.Int16,
			Instructor:        utils.PgTextToString(course.InstructorFullname),
		})
	}

	serviceLogger.Info("available courses retrieved",
		zap.Int("course_count", len(availableCourses)),
	)

	return dto.AvailableCoursesResponse{
		StudentID:        studentID,
		Department:       utils.PgTextToString(student.Department),
		ClassLevel:       student.ClassLevel.Int16,
		Semester:         semester,
		AvailableCourses: availableCourses,
	}, nil
}

// CreateEnrollmentProgram creates a new enrollment program (student submits course selection)
func (s *EnrollmentService) CreateEnrollmentProgram(ctx context.Context, req dto.CreateEnrollmentRequest) (dto.EnrollmentProgramResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "CreateEnrollmentProgram"),
		zap.String("student_id", req.StudentID.String()),
		zap.String("semester", req.Semester),
		zap.Int("course_count", len(req.CourseIDs)),
	)

	if err := validateCourseSelection(req.CourseIDs); err != nil {
		serviceLogger.Warn("course selection invalid",
			zap.Error(err),
			zap.Int("requested", len(req.CourseIDs)),
		)
		return dto.EnrollmentProgramResponse{}, err
	}

	// Enrollment uses STRICT period lock — different from grades/attendance.
	// Period inside: only students can enroll. Period outside: NOBODY can modify (admin included).
	// No hard_deadline check needed — period is the only lock.
	// Why no admin override? Enrollment is the student's own responsibility.
	// Admin should not add/remove courses on behalf of students.
	// See: docs/semester-wizard-plan.md "Ders Kayit (Enrollment) Icin: Siki Period Kilidi"
	var periodStart, periodEnd *time.Time
	period, periodErr := s.periodRepo.GetActivePeriodBySemester(ctx, req.Semester)
	if periodErr == nil {
		periodStart = &period.PeriodStart
		periodEnd = &period.PeriodEnd
	}

	enrollCheck := rules.CanEnrollInSemester(rules.EnrollmentParams{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	if !enrollCheck.Allowed {
		serviceLogger.Warn("enrollment period check failed",
			zap.String("reason", enrollCheck.Reason),
		)
		switch enrollCheck.Reason {
		case "enrollment_not_configured":
			return dto.EnrollmentProgramResponse{}, serviceErrors.ErrEnrollmentPeriodNotOpen
		case "enrollment_not_started":
			return dto.EnrollmentProgramResponse{}, serviceErrors.ErrEnrollmentPeriodNotOpen
		default:
			return dto.EnrollmentProgramResponse{}, serviceErrors.ErrEnrollmentPeriodEnded
		}
	}

	// Get student from cache
	student, err := s.studentRepo.GetStudentByID(ctx, req.StudentID)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrNotFound) {
			serviceLogger.Warn("student not found in cache")
			return dto.EnrollmentProgramResponse{}, serviceErrors.ErrStudentNotFound
		}
		return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Check if student is active
	if !student.IsActive.Bool || !student.IsActive.Valid {
		serviceLogger.Warn("student is deactivated")
		return dto.EnrollmentProgramResponse{}, serviceErrors.ErrStudentDeactivated
	}

	// Check for existing program
	existingProgram, err := s.enrollmentRepo.GetEnrollmentProgramByStudentAndSemester(ctx, req.StudentID, req.Semester)
	if err != nil && !sharedErrors.Is(err, sharedErrors.ErrNotFound) {
		return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	if existingProgram.ID.Valid {
		// If pending, auto-replace (cancel old program and create new one)
		if existingProgram.Status.EnrollmentStatusEnum == db.EnrollmentStatusEnumPending {
			serviceLogger.Info("replacing existing pending program",
				zap.String("old_program_id", utils.PgtypeToUUID(existingProgram.ID).String()),
			)

			// Get courses to decrement enrollments
			coursesRows, err := s.enrollmentRepo.GetCoursesByProgramID(ctx, utils.PgtypeToUUID(existingProgram.ID))
			if err != nil {
				return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
			}

			oldCourseIDs := make([]uuid.UUID, len(coursesRows))
			for i, row := range coursesRows {
				oldCourseIDs[i] = utils.PgtypeToUUID(row.CourseID)
			}

			cancelEventPayload := buildEnrollmentCancelledPayload(EnrollmentCancelledInputs{
				ProgramID:   utils.PgtypeToUUID(existingProgram.ID),
				StudentID:   req.StudentID,
				Semester:    req.Semester,
				CourseIDs:   oldCourseIDs,
				CancelledBy: "student",
				CancelType:  "auto_replace",
			})

			// Delete old program (with transaction)
			err = s.enrollmentRepo.CancelProgramWithEvent(ctx, utils.PgtypeToUUID(existingProgram.ID), oldCourseIDs, cancelEventPayload)
			if err != nil {
				serviceLogger.Error("failed to cancel existing program",
					zap.String("old_program_id", utils.PgtypeToUUID(existingProgram.ID).String()),
					zap.Error(err),
				)
				return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
			}

			serviceLogger.Info("existing pending program cancelled successfully, creating new program")
			// Continue with new program creation...
		} else {
			// If approved, cannot create new program
			serviceLogger.Warn("student already has an approved program for this semester",
				zap.String("existing_status", string(existingProgram.Status.EnrollmentStatusEnum)),
			)
			return dto.EnrollmentProgramResponse{}, serviceErrors.ErrAlreadySubmitted
		}
	}

	// Get courses
	courses, err := s.courseRepo.GetCoursesByIDs(ctx, req.CourseIDs)
	if err != nil {
		return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	if len(courses) != len(req.CourseIDs) {
		serviceLogger.Warn("some courses not found",
			zap.Int("requested", len(req.CourseIDs)),
			zap.Int("found", len(courses)),
		)
		return dto.EnrollmentProgramResponse{}, serviceErrors.ErrCourseNotFound
	}

	if err := validateCoursesAgainstStudent(courses, utils.PgTextToString(student.Department), student.ClassLevel.Int16); err != nil {
		serviceLogger.Warn("course list rejected against student profile",
			zap.Error(err),
			zap.String("student_dept", utils.PgTextToString(student.Department)),
			zap.Int16("student_level", student.ClassLevel.Int16),
		)
		return dto.EnrollmentProgramResponse{}, err
	}

	// Check prerequisites
	for _, course := range courses {
		if err := s.checkPrerequisites(ctx, req.StudentID, course); err != nil {
			serviceLogger.Warn("prerequisite check failed",
				zap.String("course_code", course.CourseCode),
				zap.Error(err),
			)
			return dto.EnrollmentProgramResponse{}, err
		}
	}

	// Check schedule conflicts among new courses
	conflicts, err := s.courseRepo.CheckScheduleConflict(ctx, req.CourseIDs)
	if err != nil {
		return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	if len(conflicts) > 0 {
		serviceLogger.Warn("schedule conflict detected among new courses",
			zap.Int("conflicts", len(conflicts)),
		)
		return dto.EnrollmentProgramResponse{}, serviceErrors.ErrScheduleConflict
	}

	// Check schedule conflicts with student's already approved enrollments
	existingConflicts, err := s.courseRepo.CheckScheduleConflictWithExisting(ctx, req.CourseIDs, req.StudentID)
	if err != nil {
		return dto.EnrollmentProgramResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	if len(existingConflicts) > 0 {
		serviceLogger.Warn("schedule conflict with approved enrollment",
			zap.Int("conflicts", len(existingConflicts)),
		)
		return dto.EnrollmentProgramResponse{}, serviceErrors.ErrScheduleConflict
	}

	// Check capacity and create program (with transaction)
	program, err := s.createProgramWithCapacityCheck(ctx, req, courses)
	if err != nil {
		return dto.EnrollmentProgramResponse{}, err
	}

	serviceLogger.Info("enrollment program created successfully",
		zap.String("program_id", utils.PgtypeToUUID(program.ID).String()),
	)

	return s.buildProgramResponse(ctx, program, courses), nil
}

// Helper: Check prerequisites for a course
func (s *EnrollmentService) checkPrerequisites(ctx context.Context, studentID uuid.UUID, course db.SemesterCoursesCache) error {
	// Parse prerequisites from JSONB
	var prerequisites []dto.PrerequisiteCourse
	if len(course.Prerequisites) > 0 {
		if err := json.Unmarshal(course.Prerequisites, &prerequisites); err != nil {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, fmt.Errorf("failed to parse prerequisites: %w", err))
		}
	}

	// Check each prerequisite
	for _, prereq := range prerequisites {
		passed, err := s.enrollmentRepo.CheckPrerequisitePassed(ctx, studentID, prereq.CourseCode)
		if err != nil {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		if !passed {
			return serviceErrors.ErrPrerequisitesNotMet
		}
	}

	return nil
}

// Helper: Create program with capacity check (transaction)
func (s *EnrollmentService) createProgramWithCapacityCheck(ctx context.Context, req dto.CreateEnrollmentRequest, courses []db.SemesterCoursesCache) (db.EnrollmentProgram, error) {
	// Create program parameters
	programParams := db.CreateEnrollmentProgramParams{
		StudentID: utils.UUIDToPgtype(req.StudentID),
		Semester:  req.Semester,
		Status: db.NullEnrollmentStatusEnum{
			EnrollmentStatusEnum: db.EnrollmentStatusEnumPending,
			Valid:                true,
		},
	}

	eventPayload := buildEnrollmentSubmittedPayload(EnrollmentSubmittedInputs{
		StudentID: req.StudentID,
		Semester:  req.Semester,
		CourseIDs: req.CourseIDs,
	})

	// Create program with courses and event (atomic transaction)
	program, err := s.enrollmentRepo.CreateProgramWithCoursesAndEvent(ctx, programParams, req.CourseIDs, eventPayload)
	if err != nil {
		// Check for capacity error
		if sharedErrors.Is(err, sharedErrors.ErrConflict) {
			return db.EnrollmentProgram{}, serviceErrors.ErrCourseFull
		}
		return db.EnrollmentProgram{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	return program, nil
}

// CancelMyEnrollment cancels the student's enrollment program for a semester (only if not approved)
func (s *EnrollmentService) CancelMyEnrollment(ctx context.Context, studentID uuid.UUID, semester string) error {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "EnrollmentService"),
		zap.String("method", "CancelMyEnrollment"),
		zap.String("student_id", studentID.String()),
		zap.String("semester", semester),
	)

	// Strict period lock — same as CreateEnrollmentProgram
	var periodStart, periodEnd *time.Time
	period, periodErr := s.periodRepo.GetActivePeriodBySemester(ctx, semester)
	if periodErr == nil {
		periodStart = &period.PeriodStart
		periodEnd = &period.PeriodEnd
	}
	enrollCheck := rules.CanEnrollInSemester(rules.EnrollmentParams{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	if !enrollCheck.Allowed {
		serviceLogger.Warn("enrollment period check failed for cancel",
			zap.String("reason", enrollCheck.Reason),
		)
		switch enrollCheck.Reason {
		case "enrollment_not_configured", "enrollment_not_started":
			return serviceErrors.ErrEnrollmentPeriodNotOpen
		default:
			return serviceErrors.ErrEnrollmentPeriodEnded
		}
	}

	// Get existing enrollment program
	existingProgram, err := s.enrollmentRepo.GetEnrollmentProgramByStudentAndSemester(ctx, studentID, semester)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrNotFound) {
			serviceLogger.Warn("enrollment program not found")
			return sharedErrors.WrapWithMessage(sharedErrors.ErrNotFound, err, "enrollment program not found for this semester")
		}
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Check if program exists
	if !existingProgram.ID.Valid {
		serviceLogger.Warn("no enrollment program found")
		return sharedErrors.WrapWithMessage(sharedErrors.ErrNotFound, nil, "enrollment program not found for this semester")
	}

	// Check if already approved - cannot cancel approved enrollments
	if existingProgram.Status.EnrollmentStatusEnum == db.EnrollmentStatusEnumApproved {
		serviceLogger.Warn("cannot cancel approved enrollment")
		return sharedErrors.WrapWithMessage(sharedErrors.ErrForbidden, serviceErrors.ErrCannotModifyApproved, "cannot cancel approved enrollment")
	}

	// Get courses to decrement enrollments
	coursesRows, err := s.enrollmentRepo.GetCoursesByProgramID(ctx, utils.PgtypeToUUID(existingProgram.ID))
	if err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	courseIDs := make([]uuid.UUID, len(coursesRows))
	for i, row := range coursesRows {
		courseIDs[i] = utils.PgtypeToUUID(row.CourseID)
	}

	cancelEventPayload := buildEnrollmentCancelledPayload(EnrollmentCancelledInputs{
		ProgramID:   utils.PgtypeToUUID(existingProgram.ID),
		StudentID:   studentID,
		Semester:    semester,
		CourseIDs:   courseIDs,
		CancelledBy: "student",
		CancelType:  "manual",
	})

	// Cancel program (decrement enrollments, delete program, create event)
	err = s.enrollmentRepo.CancelProgramWithEvent(ctx, utils.PgtypeToUUID(existingProgram.ID), courseIDs, cancelEventPayload)
	if err != nil {
		serviceLogger.Error("failed to cancel program", zap.Error(err))
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("enrollment program cancelled successfully",
		zap.String("program_id", utils.PgtypeToUUID(existingProgram.ID).String()),
	)

	return nil
}

// Helper: Build program response with course details
func (s *EnrollmentService) buildProgramResponse(ctx context.Context, program db.EnrollmentProgram, courses []db.SemesterCoursesCache) dto.EnrollmentProgramResponse {
	coursesDTO := make([]dto.CourseBasic, 0, len(courses))
	for _, course := range courses {
		coursesDTO = append(coursesDTO, dto.CourseBasic{
			ID:             utils.PgtypeToUUID(course.ID).String(),
			CourseCode:     course.CourseCode,
			CourseName:     utils.PgTextToString(course.CourseName),
			Credits:        course.Credits,
			InstructorName: utils.PgTextToString(course.InstructorFullname),
		})
	}

	return dto.EnrollmentProgramResponse{
		ID:        utils.PgtypeToUUID(program.ID),
		StudentID: utils.PgtypeToUUID(program.StudentID),
		Semester:  program.Semester,
		Status:    string(program.Status.EnrollmentStatusEnum),
		Courses:   coursesDTO,
		CreatedAt: program.CreatedAt.Time,
	}
}
