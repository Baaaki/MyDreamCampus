package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/db"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/dto"
	catalogErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/repository"
	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/logger"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type CatalogService struct {
	catalogRepo *repository.CatalogRepository
}

func NewCatalogService(catalogRepo *repository.CatalogRepository) *CatalogService {
	return &CatalogService{
		catalogRepo: catalogRepo,
	}
}

// CreateCourse creates a new course in the catalog
func (s *CatalogService) CreateCourse(ctx context.Context, req dto.CreateCourseRequest) (dto.CourseResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "CatalogService"),
		zap.String("method", "CreateCourse"),
		zap.String("course_code", req.CourseCode),
	)

	// Validate prerequisites
	if len(req.Prerequisites) > 0 {
		if err := s.validatePrerequisites(ctx, req.Prerequisites, req.ClassLevel); err != nil {
			return dto.CourseResponse{}, err
		}
	}

	// Convert JSON fields
	prerequisitesJSON, err := repository.PrerequisitesToJSON(req.Prerequisites)
	if err != nil {
		serviceLogger.Error("failed to convert prerequisites to JSON", zap.Error(err))
		return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	coordinatorJSON, err := repository.CoordinatorToJSON(req.Coordinator)
	if err != nil {
		serviceLogger.Error("failed to convert coordinator to JSON", zap.Error(err))
		return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	learningOutcomesListJSON, err := repository.StringSliceToJSON(req.LearningOutcomesList)
	if err != nil {
		serviceLogger.Error("failed to convert learning outcomes list to JSON", zap.Error(err))
		return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	weeklyTopicsJSON, err := repository.WeeklyTopicsToJSON(req.WeeklyTopics)
	if err != nil {
		serviceLogger.Error("failed to convert weekly topics to JSON", zap.Error(err))
		return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	recommendedSourcesJSON, err := repository.StringSliceToJSON(req.RecommendedSources)
	if err != nil {
		serviceLogger.Error("failed to convert recommended sources to JSON", zap.Error(err))
		return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Set defaults
	status := req.Status
	if status == "" {
		status = "active"
	}

	courseCategory := req.CourseCategory
	if courseCategory == "" {
		courseCategory = "theoretical"
	}

	educationLevel := req.EducationLevel
	if educationLevel == "" {
		educationLevel = "undergraduate"
	}

	teachingType := req.TeachingType
	if teachingType == "" {
		teachingType = "on_campus"
	}

	language := "Türkçe"
	if req.Language != nil {
		language = *req.Language
	}

	// Build create params
	params := db.CreateCourseParams{
		CourseCode:           req.CourseCode,
		Name:                 req.Name,
		Faculty:              req.Faculty,
		Department:           req.Department,
		OfferingUnit:         utils.PointerStringToPgText(req.OfferingUnit),
		ClassLevel:           req.ClassLevel,
		Semester:             utils.PointerInt16ToPgInt2(req.Semester),
		Credits:              req.Credits,
		Ects:                 utils.PointerInt16ToPgInt2(req.ECTS),
		TheoreticalHours:     req.TheoreticalHours,
		LabHours:             req.LabHours,
		CourseType:           db.CourseTypeEnum(req.CourseType),
		CourseCategory:       db.CourseCategoryEnum(courseCategory),
		EducationLevel:       db.EducationLevelEnum(educationLevel),
		TeachingType:         db.TeachingTypeEnum(teachingType),
		Language:             language,
		Prerequisites:        prerequisitesJSON,
		Coordinator:          coordinatorJSON,
		Purpose:              utils.PointerStringToPgText(req.Purpose),
		Description:          utils.PointerStringToPgText(req.Description),
		LearningOutcomes:     utils.PointerStringToPgText(req.LearningOutcomes),
		LearningOutcomesList: learningOutcomesListJSON,
		WeeklyTopics:         weeklyTopicsJSON,
		RecommendedSources:   recommendedSourcesJSON,
		Syllabus:             utils.PointerStringToPgText(req.Syllabus),
		Status:               db.CourseCatalogStatusEnum(status),
	}

	course, err := s.catalogRepo.CreateCourse(ctx, params)
	if err != nil {
		if sharedErrors.Is(err, catalogErrors.ErrCourseExistsRepo) {
			serviceLogger.Warn("duplicate course code detected", zap.Error(err))
			return dto.CourseResponse{}, catalogErrors.ErrCourseCodeExists
		}
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("course created successfully in catalog",
		zap.String("course_id", utils.PgtypeToUUIDString(course.ID)),
		zap.String("course_code", course.CourseCode),
		zap.String("status", string(course.Status)),
	)

	return s.toCourseResponse(course)
}

// GetCourseByCourseCode retrieves a course by its course code
func (s *CatalogService) GetCourseByCourseCode(ctx context.Context, courseCode string) (dto.CourseResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "CatalogService"),
		zap.String("method", "GetCourseByCourseCode"),
		zap.String("course_code", courseCode),
	)

	course, err := s.catalogRepo.GetCourseByCourseCode(ctx, courseCode)
	if err != nil {
		if sharedErrors.Is(err, catalogErrors.ErrCourseNotFoundRepo) {
			serviceLogger.Warn("course not found in catalog", zap.Error(err))
			return dto.CourseResponse{}, catalogErrors.ErrCourseNotFound
		}
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("course retrieved successfully from catalog",
		zap.String("course_id", utils.PgtypeToUUIDString(course.ID)),
	)

	return s.toCourseResponse(course)
}

// ListCourses lists courses with filtering and pagination
func (s *CatalogService) ListCourses(ctx context.Context, req dto.ListCoursesRequest) (dto.ListCoursesResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "CatalogService"),
		zap.String("method", "ListCourses"),
		zap.Int("page", req.Page),
		zap.Int("limit", req.Limit),
	)

	limit := int32(req.Limit)
	offset := int32((req.Page - 1) * req.Limit)

	// Build query params
	params := db.ListCoursesParams{
		Faculty:    utils.PointerStringToPgText(req.Faculty),
		Department: utils.PointerStringToPgText(req.Department),
		CourseType: db.NullCourseTypeEnum{
			CourseTypeEnum: db.CourseTypeEnum(utils.StringPointerValue(req.CourseType)),
			Valid:          req.CourseType != nil,
		},
		CourseCategory: db.NullCourseCategoryEnum{
			CourseCategoryEnum: db.CourseCategoryEnum(utils.StringPointerValue(req.CourseCategory)),
			Valid:              req.CourseCategory != nil,
		},
		EducationLevel: db.NullEducationLevelEnum{
			EducationLevelEnum: db.EducationLevelEnum(utils.StringPointerValue(req.EducationLevel)),
			Valid:              req.EducationLevel != nil,
		},
		Status: db.NullCourseCatalogStatusEnum{
			CourseCatalogStatusEnum: db.CourseCatalogStatusEnum(utils.StringPointerValue(req.Status)),
			Valid:                   req.Status != nil,
		},
		ClassLevel: pgtype.Int2{
			Int16: utils.Int16PointerValue(req.ClassLevel),
			Valid: req.ClassLevel != nil,
		},
		Semester: pgtype.Int2{
			Int16: utils.Int16PointerValue(req.Semester),
			Valid: req.Semester != nil,
		},
		Language: utils.PointerStringToPgText(req.Language),
		Search:   utils.PointerStringToPgText(req.Search),
		Offset:   offset,
		Limit:    limit,
	}

	courses, err := s.catalogRepo.ListCourses(ctx, params)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.ListCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.ListCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Count total
	countParams := db.CountCoursesParams{
		Faculty:        params.Faculty,
		Department:     params.Department,
		CourseType:     params.CourseType,
		CourseCategory: params.CourseCategory,
		EducationLevel: params.EducationLevel,
		Status:         params.Status,
		ClassLevel:     params.ClassLevel,
		Semester:       params.Semester,
		Language:       params.Language,
		Search:         params.Search,
	}

	total, err := s.catalogRepo.CountCourses(ctx, countParams)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.ListCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.ListCoursesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Convert to response DTOs
	var courseItems []dto.CourseListItem
	for _, course := range courses {
		prerequisites, err := repository.JSONToPrerequisites(course.Prerequisites)
		if err != nil {
			serviceLogger.Error("failed to parse prerequisites JSON",
				zap.Error(err),
				zap.String("course_code", course.CourseCode),
			)
			continue
		}

		courseItems = append(courseItems, dto.CourseListItem{
			ID:               uuid.UUID(course.ID.Bytes),
			CourseCode:       course.CourseCode,
			Name:             course.Name,
			Faculty:          course.Faculty,
			Department:       course.Department,
			OfferingUnit:     utils.PgTextToStringPtr(course.OfferingUnit),
			ClassLevel:       course.ClassLevel,
			Semester:         utils.PgInt2ToInt16Ptr(course.Semester),
			Credits:          course.Credits,
			ECTS:             utils.PgInt2ToInt16Ptr(course.Ects),
			TheoreticalHours: course.TheoreticalHours,
			LabHours:         course.LabHours,
			CourseType:       string(course.CourseType),
			CourseCategory:   string(course.CourseCategory),
			EducationLevel:   string(course.EducationLevel),
			TeachingType:     string(course.TeachingType),
			Language:         course.Language,
			Prerequisites:    prerequisites,
			Status:           string(course.Status),
		})
	}

	totalPages := (int(total) + req.Limit - 1) / req.Limit

	serviceLogger.Info("courses list retrieved successfully from catalog",
		zap.Int("total_records", int(total)),
		zap.Int("returned_records", len(courseItems)),
		zap.Int("total_pages", totalPages),
	)

	return dto.ListCoursesResponse{
		Data: courseItems,
		Pagination: dto.PaginationResponse{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      int(total),
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateCourse updates a course in the catalog
func (s *CatalogService) UpdateCourse(ctx context.Context, courseCode string, req dto.UpdateCourseRequest) (dto.CourseResponse, error) {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "CatalogService"),
		zap.String("method", "UpdateCourse"),
		zap.String("course_code", courseCode),
	)

	// Check if course exists
	existingCourse, err := s.catalogRepo.GetCourseByCourseCode(ctx, courseCode)
	if err != nil {
		if sharedErrors.Is(err, catalogErrors.ErrCourseNotFoundRepo) {
			serviceLogger.Warn("course not found for update", zap.Error(err))
			return dto.CourseResponse{}, catalogErrors.ErrCourseNotFound
		}
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Determine class level for prerequisite validation
	classLevel := existingCourse.ClassLevel
	if req.ClassLevel != nil {
		classLevel = *req.ClassLevel
	}

	// Validate prerequisites if provided
	var prerequisitesJSON []byte
	if req.Prerequisites != nil {
		if err := s.validatePrerequisites(ctx, *req.Prerequisites, classLevel); err != nil {
			return dto.CourseResponse{}, err
		}
		prerequisitesJSON, err = repository.PrerequisitesToJSON(*req.Prerequisites)
		if err != nil {
			serviceLogger.Error("failed to convert prerequisites to JSON", zap.Error(err))
			return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
	}

	// Convert coordinator if provided
	var coordinatorJSON []byte
	if req.Coordinator != nil {
		coordinatorJSON, err = repository.CoordinatorToJSON(req.Coordinator)
		if err != nil {
			serviceLogger.Error("failed to convert coordinator to JSON", zap.Error(err))
			return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
	}

	// Convert learning outcomes list if provided
	var learningOutcomesListJSON []byte
	if req.LearningOutcomesList != nil {
		learningOutcomesListJSON, err = repository.StringSliceToJSON(*req.LearningOutcomesList)
		if err != nil {
			serviceLogger.Error("failed to convert learning outcomes list to JSON", zap.Error(err))
			return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
	}

	// Convert weekly topics if provided
	var weeklyTopicsJSON []byte
	if req.WeeklyTopics != nil {
		weeklyTopicsJSON, err = repository.WeeklyTopicsToJSON(*req.WeeklyTopics)
		if err != nil {
			serviceLogger.Error("failed to convert weekly topics to JSON", zap.Error(err))
			return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
	}

	// Convert recommended sources if provided
	var recommendedSourcesJSON []byte
	if req.RecommendedSources != nil {
		recommendedSourcesJSON, err = repository.StringSliceToJSON(*req.RecommendedSources)
		if err != nil {
			serviceLogger.Error("failed to convert recommended sources to JSON", zap.Error(err))
			return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
	}

	// Build update params
	params := db.UpdateCourseParams{
		CourseCode:       courseCode,
		Name:             utils.PointerStringToPgText(req.Name),
		Faculty:          utils.PointerStringToPgText(req.Faculty),
		Department:       utils.PointerStringToPgText(req.Department),
		OfferingUnit:     utils.PointerStringToPgText(req.OfferingUnit),
		ClassLevel:       pgtype.Int2{Int16: utils.Int16PointerValue(req.ClassLevel), Valid: req.ClassLevel != nil},
		Semester:         pgtype.Int2{Int16: utils.Int16PointerValue(req.Semester), Valid: req.Semester != nil},
		Credits:          pgtype.Int2{Int16: utils.Int16PointerValue(req.Credits), Valid: req.Credits != nil},
		Ects:             pgtype.Int2{Int16: utils.Int16PointerValue(req.ECTS), Valid: req.ECTS != nil},
		TheoreticalHours: pgtype.Int2{Int16: utils.Int16PointerValue(req.TheoreticalHours), Valid: req.TheoreticalHours != nil},
		LabHours:         pgtype.Int2{Int16: utils.Int16PointerValue(req.LabHours), Valid: req.LabHours != nil},
		CourseType: db.NullCourseTypeEnum{
			CourseTypeEnum: db.CourseTypeEnum(utils.StringPointerValue(req.CourseType)),
			Valid:          req.CourseType != nil,
		},
		CourseCategory: db.NullCourseCategoryEnum{
			CourseCategoryEnum: db.CourseCategoryEnum(utils.StringPointerValue(req.CourseCategory)),
			Valid:              req.CourseCategory != nil,
		},
		EducationLevel: db.NullEducationLevelEnum{
			EducationLevelEnum: db.EducationLevelEnum(utils.StringPointerValue(req.EducationLevel)),
			Valid:              req.EducationLevel != nil,
		},
		TeachingType: db.NullTeachingTypeEnum{
			TeachingTypeEnum: db.TeachingTypeEnum(utils.StringPointerValue(req.TeachingType)),
			Valid:            req.TeachingType != nil,
		},
		Language:         utils.PointerStringToPgText(req.Language),
		Purpose:          utils.PointerStringToPgText(req.Purpose),
		Description:      utils.PointerStringToPgText(req.Description),
		LearningOutcomes: utils.PointerStringToPgText(req.LearningOutcomes),
		Syllabus:         utils.PointerStringToPgText(req.Syllabus),
		Status: db.NullCourseCatalogStatusEnum{
			CourseCatalogStatusEnum: db.CourseCatalogStatusEnum(utils.StringPointerValue(req.Status)),
			Valid:                   req.Status != nil,
		},
	}

	// Set JSONB fields only if provided
	if prerequisitesJSON != nil {
		params.Prerequisites = prerequisitesJSON
	}
	if coordinatorJSON != nil {
		params.Coordinator = coordinatorJSON
	}
	if learningOutcomesListJSON != nil {
		params.LearningOutcomesList = learningOutcomesListJSON
	}
	if weeklyTopicsJSON != nil {
		params.WeeklyTopics = weeklyTopicsJSON
	}
	if recommendedSourcesJSON != nil {
		params.RecommendedSources = recommendedSourcesJSON
	}

	course, err := s.catalogRepo.UpdateCourse(ctx, params)
	if err != nil {
		if sharedErrors.Is(err, catalogErrors.ErrCourseNotFoundRepo) {
			serviceLogger.Warn("course not found during update", zap.Error(err))
			return dto.CourseResponse{}, catalogErrors.ErrCourseNotFound
		}
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return dto.CourseResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	serviceLogger.Info("course updated successfully in catalog",
		zap.String("course_id", utils.PgtypeToUUIDString(course.ID)),
		zap.Bool("prerequisites_changed", req.Prerequisites != nil),
		zap.Bool("status_changed", req.Status != nil),
	)

	return s.toCourseResponse(course)
}

// validatePrerequisites validates prerequisite courses
func (s *CatalogService) validatePrerequisites(ctx context.Context, prerequisites []dto.Prerequisite, courseClassLevel int16) error {
	log := logger.WithContextAndFields(ctx,
		zap.String("service", "CatalogService"),
		zap.String("method", "validatePrerequisites"),
	)

	if len(prerequisites) == 0 {
		return nil
	}

	// Extract prerequisite IDs
	ids := make([]uuid.UUID, len(prerequisites))
	for i, prereq := range prerequisites {
		ids[i] = prereq.ID
	}

	// Fetch prerequisite courses from database
	courses, err := s.catalogRepo.GetCoursesByIDs(ctx, ids)
	if err != nil {
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Check if all prerequisites were found
	if len(courses) != len(prerequisites) {
		log.Warn("some prerequisites not found in catalog",
			zap.Int("expected", len(prerequisites)),
			zap.Int("found", len(courses)),
		)
		return catalogErrors.ErrInvalidPrerequisite
	}

	// Validate each prerequisite
	courseMap := make(map[uuid.UUID]db.GetCoursesByIDsRow)
	for _, course := range courses {
		courseMap[uuid.UUID(course.ID.Bytes)] = course
	}

	for _, prereq := range prerequisites {
		course, exists := courseMap[prereq.ID]
		if !exists {
			log.Warn("prerequisite not found",
				zap.String("prerequisite_id", prereq.ID.String()),
			)
			return catalogErrors.ErrInvalidPrerequisite
		}

		// Validate course_code matches
		if course.CourseCode != prereq.CourseCode {
			log.Warn("prerequisite course_code mismatch",
				zap.String("expected", course.CourseCode),
				zap.String("provided", prereq.CourseCode),
			)
			return catalogErrors.ErrInvalidPrerequisite
		}

		// Validate course_name matches
		if course.Name != prereq.CourseName {
			log.Warn("prerequisite course_name mismatch",
				zap.String("expected", course.Name),
				zap.String("provided", prereq.CourseName),
			)
			return catalogErrors.ErrInvalidPrerequisite
		}

		// Validate class level: prerequisite class_level must be less than course class_level
		if course.ClassLevel >= courseClassLevel {
			log.Warn("invalid prerequisite class level",
				zap.Int16("prerequisite_class_level", course.ClassLevel),
				zap.Int16("course_class_level", courseClassLevel),
			)
			return catalogErrors.ErrInvalidPrerequisiteLevel
		}
	}

	return nil
}

// toCourseResponse converts db.CourseCatalog to dto.CourseResponse
func (s *CatalogService) toCourseResponse(course db.CourseCatalog) (dto.CourseResponse, error) {
	prerequisites, err := repository.JSONToPrerequisites(course.Prerequisites)
	if err != nil {
		return dto.CourseResponse{}, fmt.Errorf("failed to parse prerequisites: %w", err)
	}

	coordinator, err := repository.JSONToCoordinator(course.Coordinator)
	if err != nil {
		return dto.CourseResponse{}, fmt.Errorf("failed to parse coordinator: %w", err)
	}

	learningOutcomesList, err := repository.JSONToStringSlice(course.LearningOutcomesList)
	if err != nil {
		return dto.CourseResponse{}, fmt.Errorf("failed to parse learning outcomes list: %w", err)
	}

	weeklyTopics, err := repository.JSONToWeeklyTopics(course.WeeklyTopics)
	if err != nil {
		return dto.CourseResponse{}, fmt.Errorf("failed to parse weekly topics: %w", err)
	}

	recommendedSources, err := repository.JSONToStringSlice(course.RecommendedSources)
	if err != nil {
		return dto.CourseResponse{}, fmt.Errorf("failed to parse recommended sources: %w", err)
	}

	return dto.CourseResponse{
		ID:                   uuid.UUID(course.ID.Bytes),
		CourseCode:           course.CourseCode,
		Name:                 course.Name,
		Faculty:              course.Faculty,
		Department:           course.Department,
		OfferingUnit:         utils.PgTextToStringPtr(course.OfferingUnit),
		ClassLevel:           course.ClassLevel,
		Semester:             utils.PgInt2ToInt16Ptr(course.Semester),
		Credits:              course.Credits,
		ECTS:                 utils.PgInt2ToInt16Ptr(course.Ects),
		TheoreticalHours:     course.TheoreticalHours,
		LabHours:             course.LabHours,
		CourseType:           string(course.CourseType),
		CourseCategory:       string(course.CourseCategory),
		EducationLevel:       string(course.EducationLevel),
		TeachingType:         string(course.TeachingType),
		Language:             course.Language,
		Prerequisites:        prerequisites,
		Coordinator:          coordinator,
		Purpose:              utils.PgTextToStringPtr(course.Purpose),
		Description:          utils.PgTextToStringPtr(course.Description),
		LearningOutcomes:     utils.PgTextToStringPtr(course.LearningOutcomes),
		LearningOutcomesList: learningOutcomesList,
		WeeklyTopics:         weeklyTopics,
		RecommendedSources:   recommendedSources,
		Syllabus:             utils.PgTextToStringPtr(course.Syllabus),
		Status:               string(course.Status),
		CreatedAt:            course.CreatedAt.Time,
		UpdatedAt:            course.UpdatedAt.Time,
	}, nil
}

// ValidateAssessmentSchema validates assessment schema structure and business rules
func ValidateAssessmentSchema(schema []dto.AssessmentItem) error {
	if len(schema) == 0 {
		return catalogErrors.ErrInvalidAssessmentSchema
	}

	// Validate slug pattern (lowercase, starts with letter, only letters/numbers/underscore)
	slugRegex := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	slugMap := make(map[string]bool)
	totalWeight := int16(0)

	for _, item := range schema {
		// Validate slug uniqueness and format
		if slugMap[item.Slug] {
			return catalogErrors.ErrDuplicateAssessmentSlug
		}
		if !slugRegex.MatchString(item.Slug) {
			return catalogErrors.ErrInvalidAssessmentSchema
		}
		slugMap[item.Slug] = true

		// Validate name (UTF-8, max 100 char)
		if item.Name == "" || len(item.Name) > 100 {
			return catalogErrors.ErrInvalidAssessmentSchema
		}

		// Validate weight (0-100)
		if item.Weight < 0 || item.Weight > 100 {
			return catalogErrors.ErrInvalidAssessmentSchema
		}

		totalWeight += item.Weight
	}

	// Validate total weight must equal 100
	if totalWeight != 100 {
		return catalogErrors.ErrAssessmentWeightNotHundred
	}

	return nil
}

// ValidateSlotNumbers validates slot numbers (must be 1-9)
func ValidateSlotNumbers(slotNumbers []int16) error {
	for _, slot := range slotNumbers {
		if slot < 1 || slot > 9 {
			return catalogErrors.ErrInvalidSlotNumber
		}
	}
	return nil
}

// ValidateDayOfWeek validates day of week enum
func ValidateDayOfWeek(day string) error {
	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
		"friday": true, "saturday": true, "sunday": true,
	}

	if !validDays[day] {
		return catalogErrors.ErrInvalidDayOfWeek
	}
	return nil
}
