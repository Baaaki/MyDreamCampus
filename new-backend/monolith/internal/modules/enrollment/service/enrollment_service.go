package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/repository"
	catalogDTO "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/dto"
	studentDTO "github.com/baaaki/mydreamcampus/monolith/internal/modules/student/dto"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/monolith/internal/platform/repository"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rules"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EnrollmentService struct {
	enrollmentRepo      *repository.EnrollmentRepository
	studentClient       StudentClient
	courseCatalogClient CourseCatalogClient
	periodRepo          *sharedRepo.SimplePeriodRepository
}

func NewEnrollmentService(
	enrollmentRepo *repository.EnrollmentRepository,
	studentClient StudentClient,
	courseCatalogClient CourseCatalogClient,
	periodRepo *sharedRepo.SimplePeriodRepository,
) *EnrollmentService {
	return &EnrollmentService{
		enrollmentRepo:      enrollmentRepo,
		studentClient:       studentClient,
		courseCatalogClient: courseCatalogClient,
		periodRepo:          periodRepo,
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
	student, err := s.studentClient.GetStudentByID(ctx, studentID)
	if err != nil {
		serviceLogger.Warn("student not found")
		return dto.AvailableCoursesResponse{}, serviceErrors.ErrStudentNotFound
	}

	// Check if student is active
	if student.Status != "active" {
		serviceLogger.Warn("student is deactivated")
		return dto.AvailableCoursesResponse{}, serviceErrors.ErrStudentDeactivated
	}

	// Get available courses
	courses, err := s.courseCatalogClient.GetAvailableCourses(ctx,
		student.Department,
		int16(student.ClassLevel),
		semester,
	)
	if err != nil {
		return dto.AvailableCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Build response
	availableCourses := make([]dto.AvailableCourse, 0, len(courses))
	for _, course := range courses {

		// Map course catalog sessions to enrollment DTO
		var scheduleSessions []dto.ScheduleSession
		for _, s := range course.ScheduleSessions {
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

		// Count active enrollments via repository to check available seats
		currentEnrollment, err := s.enrollmentRepo.CountEnrollmentForCourse(ctx, course.ID)
		if err != nil {
			serviceLogger.Error("failed to get current enrollment count", zap.Error(err))
			continue
		}

		availableCourses = append(availableCourses, dto.AvailableCourse{
			ID:                course.ID,
			CourseCode:        course.CourseCode,
			CourseName:        course.CourseName,
			Credits:           course.Credits,
			ScheduleSessions:  scheduleSessions,
			MaxCapacity:       course.MaxCapacity,
			CurrentEnrollment: int16(currentEnrollment),
			AvailableSeats:    course.MaxCapacity - int16(currentEnrollment),
			Instructor:        course.InstructorFullname,
		})
	}

	serviceLogger.Info("available courses retrieved",
		zap.Int("course_count", len(availableCourses)),
	)

	return dto.AvailableCoursesResponse{
		StudentID:        studentID,
		Department:       student.Department,
		ClassLevel:       int16(student.ClassLevel),
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
	student, err := s.studentClient.GetStudentByID(ctx, req.StudentID)
	if err != nil {
		serviceLogger.Warn("student not found")
		return dto.EnrollmentProgramResponse{}, serviceErrors.ErrStudentNotFound
	}

	// Check if student is active
	if student.Status != "active" {
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
		if existingProgram.Status.EnrollmentStatusEnum == db.EnrollmentEnrollmentStatusEnumPending {
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
			err = s.enrollmentRepo.CancelProgramWithEvent(ctx, utils.PgtypeToUUID(existingProgram.ID), cancelEventPayload)
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
	courses, err := s.courseCatalogClient.GetCoursesByIDs(ctx, req.Semester, req.CourseIDs)
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

	if err := validateCoursesAgainstStudent(courses, student.Department, int16(student.ClassLevel)); err != nil {
		serviceLogger.Warn("course list rejected against student profile",
			zap.Error(err),
			zap.String("student_dept", student.Department),
			zap.Int16("student_level", int16(student.ClassLevel)),
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
	if err := s.checkScheduleConflict(courses); err != nil {
		serviceLogger.Warn("schedule conflict detected among new courses", zap.Error(err))
		return dto.EnrollmentProgramResponse{}, serviceErrors.ErrScheduleConflict
	}

	// Check schedule conflicts with student's already approved enrollments
	statusApproved := "approved"
	approvedPrograms, err := s.enrollmentRepo.GetEnrollmentProgramsByStudent(ctx, req.StudentID, nil, &statusApproved)
	var existingCourseIDs []uuid.UUID
	if err == nil {
		for _, p := range approvedPrograms {
			cRows, _ := s.enrollmentRepo.GetCoursesByProgramID(ctx, utils.PgtypeToUUID(p.ID))
			for _, c := range cRows {
				existingCourseIDs = append(existingCourseIDs, utils.PgtypeToUUID(c.CourseID))
			}
		}
	}

	if len(existingCourseIDs) > 0 {
		existingCourses, err := s.courseCatalogClient.GetCoursesByIDs(ctx, req.Semester, existingCourseIDs)
		if err == nil {
			allCourses := append(courses, existingCourses...)
			if err := s.checkScheduleConflict(allCourses); err != nil {
				serviceLogger.Warn("schedule conflict with approved enrollment", zap.Error(err))
				return dto.EnrollmentProgramResponse{}, serviceErrors.ErrScheduleConflict
			}
		}
	}

	// Check capacity and create program (with transaction)
	program, err := s.createProgramWithCapacityCheck(ctx, req, courses, student)
	if err != nil {
		return dto.EnrollmentProgramResponse{}, err
	}

	serviceLogger.Info("enrollment program created successfully",
		zap.String("program_id", utils.PgtypeToUUID(program.ID).String()),
	)

	return s.buildProgramResponse(ctx, program, courses), nil
}

// Helper: Check prerequisites for a course
func (s *EnrollmentService) checkPrerequisites(ctx context.Context, studentID uuid.UUID, course catalogDTO.SemesterCourseResponse) error {
	// Map prerequisites
	var prerequisites []dto.PrerequisiteCourse
	for _, p := range course.Prerequisites {
		prerequisites = append(prerequisites, dto.PrerequisiteCourse{
			ID:         p.ID,
			CourseCode: p.CourseCode,
			CourseName: p.CourseName,
		})
	}

	// Check each prerequisite
	// TODO: Phase 2.x - Integrate with Grades module to check passed courses
	// For now, we bypass since student_passed_prerequisites cache was removed
	// and Grades dependency is not yet injected in Faz 2.4.
	for range prerequisites {
		passed := true // s.gradesClient.CheckPrerequisitePassed(...)
		if !passed {
			return serviceErrors.ErrPrerequisitesNotMet
		}
	}

	return nil
}

// Helper: Create program with capacity check (transaction)
func (s *EnrollmentService) createProgramWithCapacityCheck(ctx context.Context, req dto.CreateEnrollmentRequest, courses []catalogDTO.SemesterCourseResponse, student studentDTO.StudentResponse) (db.EnrollmentProgram, error) {
	// Create program parameters
	programParams := db.CreateEnrollmentProgramParams{
		StudentID:     utils.UUIDToPgtype(req.StudentID),
		Semester:      req.Semester,
		Status: db.NullEnrollmentStatusEnum{
			EnrollmentStatusEnum: db.EnrollmentEnrollmentStatusEnumPending,
			Valid:                true,
		},
	}

	eventPayload := buildEnrollmentSubmittedPayload(EnrollmentSubmittedInputs{
		StudentID: req.StudentID,
		Semester:  req.Semester,
		CourseIDs: req.CourseIDs,
	})

	// Prepare course snapshots
	var courseSnapshots []repository.ProgramCourseSnapshot
	for _, c := range courses {
		courseSnapshots = append(courseSnapshots, repository.ProgramCourseSnapshot{
			CourseID:    c.ID,
			CourseCode:  c.CourseCode,
			CourseName:  c.CourseName,
			Credits:     c.Credits,
			MaxCapacity: c.MaxCapacity,
		})
	}

	// Create program with courses and event (atomic transaction with advisory locks)
	program, err := s.enrollmentRepo.CreateProgramWithCoursesAndEvent(ctx, programParams, courseSnapshots, eventPayload)
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
	if existingProgram.Status.EnrollmentStatusEnum == db.EnrollmentEnrollmentStatusEnumApproved {
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

	// Cancel program (delete program, create event)
	err = s.enrollmentRepo.CancelProgramWithEvent(ctx, utils.PgtypeToUUID(existingProgram.ID), cancelEventPayload)
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
func (s *EnrollmentService) buildProgramResponse(ctx context.Context, program db.EnrollmentProgram, courses []catalogDTO.SemesterCourseResponse) dto.EnrollmentProgramResponse {
	coursesDTO := make([]dto.CourseBasic, 0, len(courses))
	for _, course := range courses {
		coursesDTO = append(coursesDTO, dto.CourseBasic{
			ID:             course.ID.String(),
			CourseCode:     course.CourseCode,
			CourseName:     course.CourseName,
			Credits:        course.Credits,
			InstructorName: course.InstructorFullname,
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

// Helper: Check schedule conflicts
func (s *EnrollmentService) checkScheduleConflict(courses []catalogDTO.SemesterCourseResponse) error {
	scheduleMap := make(map[string]bool)
	for _, course := range courses {
		for _, session := range course.ScheduleSessions {
			for _, slot := range session.SlotNumbers {
				key := fmt.Sprintf("%s-%d", session.DayOfWeek, slot)
				if scheduleMap[key] {
					return fmt.Errorf("schedule conflict on %s slot %d", session.DayOfWeek, slot)
				}
				scheduleMap[key] = true
			}
		}
	}
	return nil
}
