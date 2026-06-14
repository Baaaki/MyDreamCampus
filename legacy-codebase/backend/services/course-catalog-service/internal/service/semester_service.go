package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/db"
	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/dto"
	catalogErrors "github.com/baaaki/mydreamcampus/course-catalog-service/internal/errors"
	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/clock"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/repository"
	"github.com/baaaki/mydreamcampus/shared/rules"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type SemesterService struct {
	catalogRepo        *repository.CatalogRepository
	semesterRepo       *repository.SemesterRepository
	scheduleRepo       *repository.ScheduleRepository
	outboxRepo         *repository.OutboxRepository
	staffClient        StaffClient
	periodRepo         *sharedRepo.SimplePeriodRepository
	semesterStatusRepo *repository.SemesterStatusRepository
}

func NewSemesterService(
	catalogRepo *repository.CatalogRepository,
	semesterRepo *repository.SemesterRepository,
	scheduleRepo *repository.ScheduleRepository,
	outboxRepo *repository.OutboxRepository,
	staffClient StaffClient,
	periodRepo *sharedRepo.SimplePeriodRepository,
	semesterStatusRepo *repository.SemesterStatusRepository,
) *SemesterService {
	return &SemesterService{
		catalogRepo:        catalogRepo,
		semesterRepo:       semesterRepo,
		scheduleRepo:       scheduleRepo,
		outboxRepo:         outboxRepo,
		staffClient:        staffClient,
		periodRepo:         periodRepo,
		semesterStatusRepo: semesterStatusRepo,
	}
}

// validateScheduleSessionTypes validates session types against catalog hours
// If theoretical_hours == 0, theory sessions are not allowed
// If lab_hours == 0, lab sessions are not allowed
// Total theory slots must match theoretical_hours, total lab slots must match lab_hours
func validateScheduleSessionTypes(sessions []dto.ScheduleSession, theoreticalHours, labHours int16) error {
	var totalTheorySlots int16
	var totalLabSlots int16

	for _, session := range sessions {
		if session.SessionType != "theory" && session.SessionType != "lab" {
			return catalogErrors.ErrInvalidSessionType
		}

		slotCount := int16(len(session.SlotNumbers))

		if session.SessionType == "theory" {
			if theoreticalHours == 0 {
				return catalogErrors.ErrTheoryHoursZero
			}
			totalTheorySlots += slotCount
		} else {
			if labHours == 0 {
				return catalogErrors.ErrLabHoursZero
			}
			totalLabSlots += slotCount
		}
	}

	// Validate slot counts match catalog hours
	if theoreticalHours > 0 && totalTheorySlots != theoreticalHours {
		return catalogErrors.ErrTheorySlotCountMismatch
	}
	if labHours > 0 && totalLabSlots != labHours {
		return catalogErrors.ErrLabSlotCountMismatch
	}

	return nil
}

// CreateSemesterCourse creates a new semester course (manual course opening)
func (s *SemesterService) CreateSemesterCourse(ctx context.Context, semester string, req dto.CreateSemesterCourseRequest) (dto.SemesterCourseResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "SemesterService"),
		zap.String("method", "CreateSemesterCourse"),
		zap.String("semester", semester),
		zap.String("course_code", req.CourseCode),
	)

	// Validate semester format: YYYY-YYYY-Fall or YYYY-YYYY-Spring
	if !isValidSemesterFormat(semester) {
		serviceLogger.Warn("invalid semester format", zap.String("semester", semester))
		return dto.SemesterCourseResponse{}, catalogErrors.ErrInvalidSemesterFormat
	}

	// Semester course offerings can only be added while semester is in 'planned' status.
	// After activation, the course structure is FROZEN — no add/remove/modify.
	// See: docs/semester-wizard-plan.md "Iki Katmanli Degismezlik Modeli"
	if s.semesterStatusRepo != nil {
		status, err := s.semesterStatusRepo.GetSemesterStatus(ctx, semester)
		if err != nil {
			serviceLogger.Warn("semester status check failed, semester may not exist in status table",
				zap.Error(err),
			)
		} else if status != db.SemesterStatusPlanned {
			serviceLogger.Warn("semester is not in planned status — course structure is frozen",
				zap.String("semester", semester),
				zap.String("status", string(status)),
			)
			return dto.SemesterCourseResponse{}, catalogErrors.ErrSemesterCourseFrozen
		}
	}

	// Check deadline: is course creation period open?
	period, periodErr := s.periodRepo.GetActivePeriodBySemester(ctx, semester)
	if periodErr == nil {
		check := rules.IsWithinPeriod(period.PeriodStart, period.PeriodEnd)
		if !check.Allowed {
			if clock.Now().Before(period.PeriodStart) {
				return dto.SemesterCourseResponse{}, catalogErrors.ErrCourseCreationPeriodNotOpen
			}
			return dto.SemesterCourseResponse{}, catalogErrors.ErrCourseCreationPeriodEnded
		}
	}

	// Validate course_code exists and is active
	catalogCourse, err := s.catalogRepo.GetCourseByCourseCode(ctx, req.CourseCode)
	if err != nil {
		if sharedErrors.Is(err, catalogErrors.ErrCourseNotFoundRepo) {
			serviceLogger.Warn("course not found in catalog",
				zap.Error(err),
			)
			return dto.SemesterCourseResponse{}, catalogErrors.ErrCourseNotFound
		}

		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Check if course is active
	if catalogCourse.Status != db.CourseCatalogStatusEnumActive {
		serviceLogger.Warn("course is not active",
			zap.String("status", string(catalogCourse.Status)),
		)
		return dto.SemesterCourseResponse{}, catalogErrors.ErrCourseNotActive
	}

	// Check credits > 0 (semester_courses requires credits > 0)
	if catalogCourse.Credits <= 0 {
		serviceLogger.Warn("course has 0 credits in catalog",
			zap.String("course_code", req.CourseCode),
		)
		return dto.SemesterCourseResponse{}, catalogErrors.ErrCourseCreditsZero
	}

	// Check class_level consistency
	if catalogCourse.ClassLevel != req.ClassLevel {
		serviceLogger.Warn("class level mismatch",
			zap.Int16("catalog_class_level", catalogCourse.ClassLevel),
			zap.Int16("request_class_level", req.ClassLevel),
		)
		return dto.SemesterCourseResponse{}, catalogErrors.ErrClassLevelMismatch
	}

	// Note: We don't check if course already exists here to avoid race condition
	// Instead, we rely on database UNIQUE(semester, course_code) constraint
	// The constraint violation will be caught during insert

	// Validate instructor and get actual fullname
	instructor, err := s.staffClient.GetInstructor(ctx, req.InstructorID, catalogCourse.Department)
	if err != nil {
		return dto.SemesterCourseResponse{}, err
	}

	// Validate slot numbers and day of week
	for _, session := range req.ScheduleSessions {
		if err := ValidateSlotNumbers(session.SlotNumbers); err != nil {
			return dto.SemesterCourseResponse{}, err
		}
		if err := ValidateDayOfWeek(session.DayOfWeek); err != nil {
			return dto.SemesterCourseResponse{}, err
		}
	}

	// Validate session types against catalog hours
	if err := validateScheduleSessionTypes(req.ScheduleSessions, catalogCourse.TheoreticalHours, catalogCourse.LabHours); err != nil {
		return dto.SemesterCourseResponse{}, err
	}

	// Apply default assessment schema if not provided
	if len(req.AssessmentSchema) == 0 {
		req.AssessmentSchema = []dto.AssessmentItem{
			{Slug: "midterm", Name: "Vize", Weight: 40},
			{Slug: "final", Name: "Final", Weight: 60},
		}
	}

	// Validate assessment schema
	if err := ValidateAssessmentSchema(req.AssessmentSchema); err != nil {
		return dto.SemesterCourseResponse{}, err
	}

	// Check instructor schedule conflict
	if err := s.checkInstructorConflict(ctx, semester, req.InstructorID, req.ScheduleSessions, uuid.Nil, catalogCourse.Department); err != nil {
		return dto.SemesterCourseResponse{}, err
	}

	// Convert assessment schema to JSONB
	assessmentSchemaJSON, err := repository.AssessmentSchemaToJSON(req.AssessmentSchema)
	if err != nil {
		serviceLogger.Error("failed to convert assessment schema to JSON",
			zap.Error(err),
		)
		return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Parse and snapshot prerequisites from catalog
	prerequisites, err := repository.JSONToPrerequisites(catalogCourse.Prerequisites)
	if err != nil {
		serviceLogger.Error("failed to parse prerequisites",
			zap.Error(err),
		)
		return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Convert prerequisites to JSONB for snapshot
	prerequisitesJSON, err := repository.PrerequisitesToJSON(prerequisites)
	if err != nil {
		serviceLogger.Error("failed to convert prerequisites to JSON",
			zap.Error(err),
		)
		return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Begin transaction
	tx, err := s.outboxRepo.BeginTx(ctx)
	if err != nil {
		serviceLogger.Error("failed to begin transaction", zap.Error(err))
		return dto.SemesterCourseResponse{}, catalogErrors.ErrTransactionFailed
	}
	defer tx.Rollback(ctx)

	// Get transaction-aware repositories
	semesterRepoTx := s.semesterRepo.WithTx(tx)
	scheduleRepoTx := s.scheduleRepo.WithTx(tx)

	// Create semester course
	semesterCourseParams := db.CreateSemesterCourseParams{
		Semester:           semester,
		CourseCode:         req.CourseCode,
		Department:         catalogCourse.Department,
		Credits:            catalogCourse.Credits,
		ClassLevel:         req.ClassLevel,
		InstructorID:       utils.UUIDToPgtype(req.InstructorID),
		InstructorFullname: instructor.FullName,
		ClassroomLocation:  req.ClassroomLocation,
		MaxCapacity:        req.MaxCapacity,
		AssessmentSchema:   assessmentSchemaJSON,
		Prerequisites:      prerequisitesJSON,
	}

	semesterCourse, err := semesterRepoTx.CreateSemesterCourse(ctx, semesterCourseParams)
	if err != nil {
		// Check if course already opened (unique constraint violation)
		if sharedErrors.Is(err, catalogErrors.ErrCourseAlreadyOpenedRepo) {
			serviceLogger.Warn("course already opened for this semester (race condition caught)",
				zap.Error(err),
			)
			return dto.SemesterCourseResponse{}, catalogErrors.ErrCourseAlreadyOpened
		}

		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Create schedule sessions (bulk insert for performance)
	var sessionParams []db.CreateScheduleSessionParams
	for _, session := range req.ScheduleSessions {
		for _, slotNumber := range session.SlotNumbers {
			sessionParams = append(sessionParams, db.CreateScheduleSessionParams{
				SemesterCourseID: semesterCourse.ID,
				DayOfWeek:        db.DayOfWeekEnum(session.DayOfWeek),
				SlotNumber:       slotNumber,
				SessionType:      db.ScheduleSessionTypeEnum(session.SessionType),
			})
		}
	}

	if err := scheduleRepoTx.BulkCreateScheduleSessions(ctx, sessionParams); err != nil {
		return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		serviceLogger.Error("failed to commit transaction",
			zap.Error(err),
		)
		return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, fmt.Errorf("transaction commit failed: %w", err))
	}

	serviceLogger.Info("semester course created successfully",
		zap.String("semester_course_id", utils.PgtypeToUUIDString(semesterCourse.ID)),
		zap.String("instructor_id", req.InstructorID.String()),
	)

	return s.toSemesterCourseResponse(semesterCourse, catalogCourse.Name, req.ScheduleSessions, prerequisites)
}

// GetSemesterCourseByID retrieves a semester course by ID
func (s *SemesterService) GetSemesterCourseByID(ctx context.Context, semester, courseID string) (dto.SemesterCourseResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "SemesterService"),
		zap.String("method", "GetSemesterCourseByID"),
		zap.String("semester", semester),
		zap.String("course_id", courseID),
	)

	// Parse course ID
	id, err := uuid.Parse(courseID)
	if err != nil {
		serviceLogger.Warn("invalid course ID format",
			zap.Error(err),
		)
		return dto.SemesterCourseResponse{}, sharedErrors.ErrInvalidID
	}

	// Get semester course
	semesterCourse, err := s.semesterRepo.GetSemesterCourseByID(ctx, id, semester)
	if err != nil {
		if sharedErrors.Is(err, catalogErrors.ErrSemesterCourseNotFoundRepo) {
			serviceLogger.Warn("semester course not found",
				zap.Error(err),
			)
			return dto.SemesterCourseResponse{}, catalogErrors.ErrSemesterCourseNotFound
		}

		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Get catalog course for course name
	catalogCourse, err := s.catalogRepo.GetCourseByCourseCode(ctx, semesterCourse.CourseCode)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Get schedule sessions
	scheduleSessions, err := s.scheduleRepo.GetScheduleSessionsByCourseID(ctx, id)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.SemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Group schedule sessions by day and session type
	scheduleSessionsDTO := groupScheduleSessionsFromRows(scheduleSessions)

	// Parse prerequisites from snapshot (not from catalog - historical data)
	prerequisites, err := repository.JSONToPrerequisites(semesterCourse.Prerequisites)
	if err != nil {
		serviceLogger.Error("failed to parse prerequisites from snapshot",
			zap.Error(err),
		)
		prerequisites = []dto.Prerequisite{}
	}

	serviceLogger.Info("semester course retrieved successfully",
		zap.String("course_code", semesterCourse.CourseCode),
	)

	return s.toSemesterCourseResponseWithPrerequisites(semesterCourse, catalogCourse.Name, scheduleSessionsDTO, prerequisites)
}

// ListSemesterCourses lists semester courses with filtering and pagination
func (s *SemesterService) ListSemesterCourses(ctx context.Context, semester string, req dto.ListSemesterCoursesRequest) (dto.ListSemesterCoursesResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "SemesterService"),
		zap.String("method", "ListSemesterCourses"),
		zap.String("semester", semester),
		zap.Int("page", req.Page),
		zap.Int("limit", req.Limit),
	)

	limit := int32(req.Limit)
	offset := int32((req.Page - 1) * req.Limit)

	// Build query params
	params := db.ListSemesterCoursesParams{
		Semester:     semester,
		Faculty:      utils.PointerStringToPgText(req.Faculty),
		Department:   utils.PointerStringToPgText(req.Department),
		InstructorID: utils.PointerUUIDToPgtype(req.InstructorID),
		CourseType: db.NullCourseTypeEnum{
			CourseTypeEnum: db.CourseTypeEnum(utils.StringPointerValue(req.CourseType)),
			Valid:          req.CourseType != nil,
		},
		ClassLevel: pgtype.Int2{
			Int16: utils.Int16PointerValue(req.ClassLevel),
			Valid: req.ClassLevel != nil,
		},
		Offset: offset,
		Limit:  limit,
	}

	courses, err := s.semesterRepo.ListSemesterCourses(ctx, params)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.ListSemesterCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.ListSemesterCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Count total
	countParams := db.CountSemesterCoursesParams{
		Semester:     params.Semester,
		Faculty:      params.Faculty,
		Department:   params.Department,
		InstructorID: params.InstructorID,
		CourseType:   params.CourseType,
		ClassLevel:   params.ClassLevel,
	}

	total, err := s.semesterRepo.CountSemesterCourses(ctx, countParams)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.ListSemesterCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.ListSemesterCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Collect all course IDs for batch schedule fetching (prevents N+1 query)
	courseIDs := make([]uuid.UUID, len(courses))
	for i, course := range courses {
		courseIDs[i] = uuid.UUID(course.ID.Bytes)
	}

	// Fetch all schedule sessions in one query
	allScheduleSessions, err := s.scheduleRepo.GetScheduleSessionsByMultipleCourseIDs(ctx, courseIDs)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.ListSemesterCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.ListSemesterCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Group schedule sessions by course ID
	scheduleMap := make(map[uuid.UUID][]db.GetScheduleSessionsByMultipleCourseIDsRow)
	for _, session := range allScheduleSessions {
		courseID := uuid.UUID(session.SemesterCourseID.Bytes)
		scheduleMap[courseID] = append(scheduleMap[courseID], session)
	}

	// Convert to response DTOs
	var courseItems []dto.SemesterCourseListItem
	for _, course := range courses {
		courseID := uuid.UUID(course.ID.Bytes)

		// Get schedule sessions from map (already fetched)
		scheduleSessions := scheduleMap[courseID]

		// Group schedule sessions
		scheduleSessionsDTO := groupScheduleSessionsFromMultiRows(scheduleSessions)

		// Parse assessment schema
		assessmentSchema, err := repository.JSONToAssessmentSchema(course.AssessmentSchema)
		if err != nil {
			serviceLogger.Error("failed to parse assessment schema",
				zap.Error(err),
				zap.String("course_id", utils.PgtypeToUUIDString(course.ID)),
			)
			continue
		}

		courseItems = append(courseItems, dto.SemesterCourseListItem{
			ID:                 uuid.UUID(course.ID.Bytes),
			Semester:           course.Semester,
			CourseCode:         course.CourseCode,
			CourseName:         course.CourseName,
			Department:         course.Department,
			Credits:            course.Credits,
			ClassLevel:         course.ClassLevel,
			InstructorID:       uuid.UUID(course.InstructorID.Bytes),
			InstructorFullname: course.InstructorFullname,
			ClassroomLocation:  course.ClassroomLocation,
			MaxCapacity:        course.MaxCapacity,
			AssessmentSchema:   assessmentSchema,
			ScheduleSessions:   scheduleSessionsDTO,
		})
	}

	totalPages := (int(total) + req.Limit - 1) / req.Limit

	serviceLogger.Info("semester courses list retrieved successfully",
		zap.Int("total_records", int(total)),
		zap.Int("returned_records", len(courseItems)),
		zap.Int("total_pages", totalPages),
	)

	return dto.ListSemesterCoursesResponse{
		Data: courseItems,
		Pagination: dto.PaginationResponse{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      int(total),
			TotalPages: totalPages,
		},
	}, nil
}

// DeleteSemesterCourse deletes a semester course
func (s *SemesterService) DeleteSemesterCourse(ctx context.Context, semester, courseID string) (dto.DeleteSemesterCourseResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "SemesterService"),
		zap.String("method", "DeleteSemesterCourse"),
		zap.String("semester", semester),
		zap.String("course_id", courseID),
	)

	// Parse course ID
	id, err := uuid.Parse(courseID)
	if err != nil {
		serviceLogger.Warn("invalid course ID format",
			zap.Error(err),
		)
		return dto.DeleteSemesterCourseResponse{}, sharedErrors.ErrInvalidID
	}

	// Semester course offerings can only be deleted while semester is in 'planned' status.
	if s.semesterStatusRepo != nil {
		status, statusErr := s.semesterStatusRepo.GetSemesterStatus(ctx, semester)
		if statusErr == nil && status != db.SemesterStatusPlanned {
			serviceLogger.Warn("semester is not in planned status — course structure is frozen")
			return dto.DeleteSemesterCourseResponse{}, catalogErrors.ErrSemesterCourseFrozen
		}
	}

	// Get existing semester course
	existingCourse, err := s.semesterRepo.GetSemesterCourseByID(ctx, id, semester)
	if err != nil {
		if sharedErrors.Is(err, catalogErrors.ErrSemesterCourseNotFoundRepo) {
			serviceLogger.Warn("semester course not found for deletion",
				zap.Error(err),
			)
			return dto.DeleteSemesterCourseResponse{}, catalogErrors.ErrSemesterCourseNotFound
		}

		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.DeleteSemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}

		return dto.DeleteSemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Delete semester course (CASCADE will delete schedule sessions)
	if err := s.semesterRepo.DeleteSemesterCourse(ctx, id); err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.DeleteSemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.DeleteSemesterCourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("semester course deleted successfully",
		zap.String("course_code", existingCourse.CourseCode),
	)

	return dto.DeleteSemesterCourseResponse{
		Message:          "Semester course deleted successfully",
		SemesterCourseID: courseID,
		CourseCode:       existingCourse.CourseCode,
		Semester:         semester,
	}, nil
}

// checkInstructorConflict checks if instructor has schedule conflict
// currentDepartment is used to detect cross-department conflicts and return a detailed error
func (s *SemesterService) checkInstructorConflict(ctx context.Context, semester string, instructorID uuid.UUID, sessions []dto.ScheduleSession, excludeCourseID uuid.UUID, currentDepartment string) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("service", "SemesterService"),
		zap.String("method", "checkInstructorConflict"),
	)

	// Build parallel arrays of (day, slot) pairs for tuple matching
	var dayEnums []db.DayOfWeekEnum
	var slots []int16

	for _, session := range sessions {
		for _, slotNumber := range session.SlotNumbers {
			dayEnums = append(dayEnums, db.DayOfWeekEnum(session.DayOfWeek))
			slots = append(slots, slotNumber)
		}
	}

	params := db.CheckInstructorScheduleConflictParams{
		Days:            dayEnums,
		Slots:           slots,
		Semester:        semester,
		InstructorID:    utils.UUIDToPgtype(instructorID),
		ExcludeCourseID: utils.UUIDToPgtype(excludeCourseID),
	}

	conflicts, err := s.scheduleRepo.CheckInstructorScheduleConflict(ctx, params)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Check for cross-department conflict first
	for _, c := range conflicts {
		if c.Department != currentDepartment {
			log.Warn("instructor has cross-department schedule conflict",
				zap.String("instructor_id", instructorID.String()),
				zap.String("conflicting_course", c.CourseCode),
				zap.String("conflicting_department", c.Department),
				zap.String("current_department", currentDepartment),
			)
			return catalogErrors.NewScheduleConflictError(c.CourseCode, c.Department, string(c.DayOfWeek), c.SlotNumber)
		}
	}

	// Same department conflict
	if len(conflicts) > 0 {
		log.Warn("instructor has schedule conflict",
			zap.String("instructor_id", instructorID.String()),
			zap.String("conflicting_course", conflicts[0].CourseCode),
		)
		return catalogErrors.ErrInstructorScheduleConflict
	}

	return nil
}

// scheduleGroupKey is used as map key for grouping sessions by day + session_type
type scheduleGroupKey struct {
	DayOfWeek   string
	SessionType string
}

// groupScheduleSessionsFromRows groups schedule sessions from GetScheduleSessionsByCourseIDRow by day and session type
func groupScheduleSessionsFromRows(sessions []db.GetScheduleSessionsByCourseIDRow) []dto.ScheduleSession {
	dayTypeMap := make(map[scheduleGroupKey][]int16)

	for _, session := range sessions {
		key := scheduleGroupKey{
			DayOfWeek:   string(session.DayOfWeek),
			SessionType: string(session.SessionType),
		}
		dayTypeMap[key] = append(dayTypeMap[key], session.SlotNumber)
	}

	var result []dto.ScheduleSession
	for key, slots := range dayTypeMap {
		result = append(result, dto.ScheduleSession{
			DayOfWeek:   key.DayOfWeek,
			SlotNumbers: slots,
			SessionType: key.SessionType,
		})
	}

	return result
}

// groupScheduleSessionsFromMultiRows groups schedule sessions from GetScheduleSessionsByMultipleCourseIDsRow by day and session type
func groupScheduleSessionsFromMultiRows(sessions []db.GetScheduleSessionsByMultipleCourseIDsRow) []dto.ScheduleSession {
	dayTypeMap := make(map[scheduleGroupKey][]int16)

	for _, session := range sessions {
		key := scheduleGroupKey{
			DayOfWeek:   string(session.DayOfWeek),
			SessionType: string(session.SessionType),
		}
		dayTypeMap[key] = append(dayTypeMap[key], session.SlotNumber)
	}

	var result []dto.ScheduleSession
	for key, slots := range dayTypeMap {
		result = append(result, dto.ScheduleSession{
			DayOfWeek:   key.DayOfWeek,
			SlotNumbers: slots,
			SessionType: key.SessionType,
		})
	}

	return result
}

// toSemesterCourseResponse converts db.SemesterCourse to dto.SemesterCourseResponse
func (s *SemesterService) toSemesterCourseResponse(course db.SemesterCourse, courseName string, sessions []dto.ScheduleSession, prerequisites []dto.Prerequisite) (dto.SemesterCourseResponse, error) {
	assessmentSchema, err := repository.JSONToAssessmentSchema(course.AssessmentSchema)
	if err != nil {
		return dto.SemesterCourseResponse{}, fmt.Errorf("failed to parse assessment schema: %w", err)
	}

	return dto.SemesterCourseResponse{
		ID:                 uuid.UUID(course.ID.Bytes),
		Semester:           course.Semester,
		CourseCode:         course.CourseCode,
		CourseName:         courseName,
		Department:         course.Department,
		Credits:            course.Credits,
		ClassLevel:         course.ClassLevel,
		InstructorID:       uuid.UUID(course.InstructorID.Bytes),
		InstructorFullname: course.InstructorFullname,
		ClassroomLocation:  course.ClassroomLocation,
		MaxCapacity:        course.MaxCapacity,
		AssessmentSchema:   assessmentSchema,
		ScheduleSessions:   sessions,
		Prerequisites:      prerequisites,
		CreatedAt:          course.CreatedAt.Time,
		UpdatedAt:          course.UpdatedAt.Time,
	}, nil
}

// toSemesterCourseResponseWithPrerequisites is alias for toSemesterCourseResponse
func (s *SemesterService) toSemesterCourseResponseWithPrerequisites(course db.SemesterCourse, courseName string, sessions []dto.ScheduleSession, prerequisites []dto.Prerequisite) (dto.SemesterCourseResponse, error) {
	return s.toSemesterCourseResponse(course, courseName, sessions, prerequisites)
}

// GetTeacherCourses returns all courses for a specific instructor
func (s *SemesterService) GetTeacherCourses(ctx context.Context, instructorID uuid.UUID, semester *string) (dto.TeacherCoursesResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "SemesterService"),
		zap.String("method", "GetTeacherCourses"),
		zap.String("instructor_id", instructorID.String()),
	)

	serviceLogger.Info("getting teacher courses")

	var semesterArg pgtype.Text
	if semester != nil && *semester != "" {
		semesterArg = pgtype.Text{String: *semester, Valid: true}
	}

	courses, err := s.semesterRepo.GetTeacherCourses(ctx, instructorID, semesterArg)
	if err != nil {
		serviceLogger.Error("failed to get teacher courses", zap.Error(err))
		return dto.TeacherCoursesResponse{}, sharedErrors.ErrInternal
	}

	// Build course items with schedule sessions
	var courseItems []dto.TeacherCourseItem
	for _, course := range courses {
		// Get schedule sessions for this course
		sessions, _ := s.scheduleRepo.GetScheduleSessionsByCourseID(ctx, utils.PgUUIDToUUID(course.ID))

		var scheduleSessions []dto.TeacherScheduleSession
		for _, session := range sessions {
			// Convert slot number to time (each slot is typically 50 min, starting from 08:00)
			startHour := 8 + (int(session.SlotNumber)-1)/2
			startMin := ((int(session.SlotNumber) - 1) % 2) * 50
			endHour := startHour
			endMin := startMin + 50
			if endMin >= 60 {
				endHour++
				endMin -= 60
			}

			scheduleSessions = append(scheduleSessions, dto.TeacherScheduleSession{
				Day:         string(session.DayOfWeek),
				Time:        fmt.Sprintf("%02d:%02d-%02d:%02d", startHour, startMin, endHour, endMin),
				Room:        course.ClassroomLocation,
				SessionType: string(session.SessionType),
			})
		}

		courseItems = append(courseItems, dto.TeacherCourseItem{
			ID:                utils.PgUUIDToUUID(course.ID),
			CourseCode:        course.CourseCode,
			CourseName:        course.CourseName,
			Faculty:           course.Faculty,
			Department:        course.Department,
			Semester:          course.Semester,
			Credits:           course.Credits,
			TheoreticalHours:  course.TheoreticalHours,
			LabHours:          course.LabHours,
			ClassroomLocation: course.ClassroomLocation,
			MaxCapacity:       course.MaxCapacity,
			Schedule:          scheduleSessions,
		})
	}

	serviceLogger.Info("teacher courses retrieved",
		zap.Int("count", len(courseItems)),
	)

	return dto.TeacherCoursesResponse{
		InstructorID: instructorID,
		TotalCourses: len(courseItems),
		Courses:      courseItems,
	}, nil
}

// semesterFormatRegex validates semester format: YYYY-YYYY-Fall or YYYY-YYYY-Spring
var semesterFormatRegex = regexp.MustCompile(`^(\d{4})-(\d{4})-(Fall|Spring)$`)

// isValidSemesterFormat checks if a semester string matches the expected format
// and the second year is exactly one more than the first (e.g., 2025-2026)
func isValidSemesterFormat(semester string) bool {
	matches := semesterFormatRegex.FindStringSubmatch(semester)
	if matches == nil {
		return false
	}

	startYear := 0
	endYear := 0
	fmt.Sscanf(matches[1], "%d", &startYear)
	fmt.Sscanf(matches[2], "%d", &endYear)

	if endYear != startYear+1 {
		return false
	}
	if startYear < 2000 || startYear > 2100 {
		return false
	}

	return true
}
