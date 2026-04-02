package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/baaaki/mydreamcampus/student-service/internal/db"
	"github.com/baaaki/mydreamcampus/student-service/internal/dto"
	"github.com/baaaki/mydreamcampus/student-service/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ImportService struct {
	importRepo   *repository.ImportRepository
	studentRepo  *repository.StudentRepository
	staffService StaffServiceInterface
}

func NewImportService(
	importRepo *repository.ImportRepository,
	studentRepo *repository.StudentRepository,
	staffService StaffServiceInterface,
) *ImportService {
	return &ImportService{
		importRepo:   importRepo,
		studentRepo:  studentRepo,
		staffService: staffService,
	}
}

// BulkImportStudents handles CSV bulk import
func (s *ImportService) BulkImportStudents(ctx context.Context, fileName string, fileReader io.Reader, createdBy uuid.UUID) (string, error) {
	// Parse CSV
	csvReader := csv.NewReader(fileReader)
	csvReader.TrimLeadingSpace = true

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		logger.Error("failed to read CSV header",
			zap.Error(err),
		)
		return "", errors.ErrValidation
	}

	// Validate header
	expectedHeader := []string{"student_number", "first_name", "last_name", "email", "faculty", "department", "enrollment_year", "class_level"}
	if !validateHeader(header, expectedHeader) {
		logger.Error("invalid CSV header",
			zap.Strings("expected", expectedHeader),
			zap.Strings("got", header),
		)
		return "", errors.ErrValidation
	}

	// Read all records
	var students []db.CreateStudentParams
	var importErrors []string
	lineNumber := 1 // Start from 1 (header is line 0)

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("failed to read CSV record",
				zap.Error(err),
				zap.Int("line", lineNumber+1),
			)
			importErrors = append(importErrors, fmt.Sprintf("Line %d: %v", lineNumber+1, err))
			lineNumber++
			continue
		}

		// Parse student record
		student, parseErr := parseStudentRecord(record, lineNumber+1)
		if parseErr != nil {
			importErrors = append(importErrors, parseErr.Error())
			lineNumber++
			continue
		}

		students = append(students, student)
		lineNumber++
	}

	totalRecords := len(students) + len(importErrors)

	// Create import job
	job, err := s.importRepo.CreateImportJob(ctx, db.CreateImportJobParams{
		FileName:     fileName,
		TotalRecords: int32(totalRecords),
		CreatedBy:    utils.UUIDToPgtype(createdBy),
	})
	if err != nil {
		logger.Error("failed to create import job",
			zap.Error(err),
		)
		return "", errors.ErrInternal
	}

	jobID := utils.PgtypeToUUIDString(job.ID)

	// Start background processing
	go s.processImport(context.Background(), utils.PgtypeToUUID(job.ID), students, importErrors)

	logger.Info("bulk import job created",
		zap.String("job_id", jobID),
		zap.Int("total_records", totalRecords),
		zap.Int("valid_records", len(students)),
	)

	return jobID, nil
}

// processImport processes the import in background
func (s *ImportService) processImport(ctx context.Context, jobID uuid.UUID, students []db.CreateStudentParams, initialErrors []string) {
	// Mark job as processing
	if err := s.importRepo.StartImportJob(ctx, jobID); err != nil {
		logger.Error("failed to start import job",
			zap.Error(err),
			zap.String("job_id", jobID.String()),
		)
		return
	}

	logger.Info("starting import processing",
		zap.String("job_id", jobID.String()),
		zap.Int("student_count", len(students)),
	)

	// Auto-assign advisors using round-robin
	if err := s.autoAssignAdvisors(ctx, students); err != nil {
		logger.Error("failed to auto-assign advisors",
			zap.Error(err),
		)
		// Continue without advisors
	}

	// Bulk insert using PostgreSQL COPY
	successCount := 0
	failCount := len(initialErrors)
	allErrors := initialErrors

	if len(students) > 0 {
		if err := s.importRepo.BulkInsertStudents(ctx, students); err != nil {
			logger.Error("bulk insert failed",
				zap.Error(err),
				zap.String("job_id", jobID.String()),
			)
			failCount += len(students)
			allErrors = append(allErrors, fmt.Sprintf("Bulk insert failed: %v", err))
		} else {
			successCount = len(students)
			logger.Info("bulk insert successful",
				zap.Int("count", successCount),
				zap.String("job_id", jobID.String()),
			)
		}
	}

	// Update job progress
	errorsJSON := []byte("[]")
	if len(allErrors) > 0 {
		errorsJSON = []byte(fmt.Sprintf(`["%s"]`, strings.Join(allErrors, `","`)))
	}

	if err := s.importRepo.UpdateImportJobProgress(ctx, db.UpdateImportJobProgressParams{
		ID:                utils.UUIDToPgtype(jobID),
		ProcessedRecords:  int32(successCount + failCount),
		SuccessfulRecords: int32(successCount),
		FailedRecords:     int32(failCount),
		Errors:            errorsJSON,
	}); err != nil {
		logger.Error("failed to update job progress",
			zap.Error(err),
		)
	}

	// Mark job as completed
	if failCount > 0 && successCount == 0 {
		s.importRepo.FailImportJob(ctx, jobID)
	} else {
		s.importRepo.CompleteImportJob(ctx, jobID)
	}

	logger.Info("import processing completed",
		zap.String("job_id", jobID.String()),
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
	)
}

// autoAssignAdvisors assigns advisors to students using round-robin
func (s *ImportService) autoAssignAdvisors(ctx context.Context, students []db.CreateStudentParams) error {
	// Group students by department
	departmentMap := make(map[string][]int)
	for i, student := range students {
		departmentMap[student.Department] = append(departmentMap[student.Department], i)
	}

	// For each department, get instructors and assign round-robin
	for department, indices := range departmentMap {
		// Get instructors for this department from staff service
		instructors, err := s.staffService.GetInstructorsByDepartment(ctx, department)
		if err != nil {
			logger.Warn("failed to get instructors for department",
				zap.String("department", department),
				zap.Error(err),
			)
			continue
		}

		if len(instructors) == 0 {
			logger.Warn("no instructors found for department",
				zap.String("department", department),
			)
			continue
		}

		// Round-robin assignment
		for i, studentIdx := range indices {
			instructorIdx := i % len(instructors)
			students[studentIdx].AdvisorID = utils.UUIDToPgtype(instructors[instructorIdx])
		}

		logger.Info("assigned advisors for department",
			zap.String("department", department),
			zap.Int("student_count", len(indices)),
			zap.Int("instructor_count", len(instructors)),
		)
	}

	return nil
}

// GetImportJobStatus retrieves import job status
func (s *ImportService) GetImportJobStatus(ctx context.Context, jobID string) (dto.ImportJobResponse, error) {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return dto.ImportJobResponse{}, errors.ErrInvalidID
	}

	job, err := s.importRepo.GetImportJobByID(ctx, id)
	if err != nil {
		logger.Error("failed to get import job",
			zap.Error(err),
			zap.String("job_id", jobID),
		)
		return dto.ImportJobResponse{}, errors.ErrNotFound
	}

	return s.toImportJobResponse(job), nil
}

// ListImportJobs lists import jobs for a user
func (s *ImportService) ListImportJobs(ctx context.Context, userID uuid.UUID, query dto.ImportJobFilterQuery) (dto.ImportJobListResponse, error) {
	limit := int32(20)
	offset := int32(0)

	if query.Limit > 0 {
		limit = int32(query.Limit)
	}
	if query.Page > 0 {
		offset = int32((query.Page - 1) * query.Limit)
	}

	jobs, total, err := s.importRepo.ListImportJobsByUser(ctx, userID, limit, offset)
	if err != nil {
		logger.Error("failed to list import jobs",
			zap.Error(err),
		)
		return dto.ImportJobListResponse{}, errors.ErrInternal
	}

	var jobSummaries []dto.ImportJobSummary
	for _, job := range jobs {
		jobSummaries = append(jobSummaries, s.toImportJobSummary(job))
	}

	return dto.ImportJobListResponse{
		Data: jobSummaries,
		Pagination: dto.PaginationResponse{
			Page:       query.Page,
			Limit:      query.Limit,
			Total:      int(total),
			TotalPages: (int(total) + query.Limit - 1) / query.Limit,
		},
	}, nil
}

// Helper functions

func validateHeader(got, expected []string) bool {
	if len(got) != len(expected) {
		return false
	}
	for i, col := range expected {
		if strings.TrimSpace(got[i]) != col {
			return false
		}
	}
	return true
}

func parseStudentRecord(record []string, lineNumber int) (db.CreateStudentParams, error) {
	if len(record) != 8 {
		return db.CreateStudentParams{}, fmt.Errorf("Line %d: expected 8 columns, got %d", lineNumber, len(record))
	}

	// Parse enrollment year
	enrollmentYear, err := strconv.Atoi(strings.TrimSpace(record[6]))
	if err != nil {
		return db.CreateStudentParams{}, fmt.Errorf("Line %d: invalid enrollment_year: %v", lineNumber, err)
	}

	// Parse class level
	classLevel, err := strconv.ParseInt(strings.TrimSpace(record[7]), 10, 16)
	if err != nil {
		return db.CreateStudentParams{}, fmt.Errorf("Line %d: invalid class_level: %v", lineNumber, err)
	}

	if classLevel < 1 || classLevel > 6 {
		return db.CreateStudentParams{}, fmt.Errorf("Line %d: class_level must be between 1 and 6", lineNumber)
	}

	return db.CreateStudentParams{
		StudentNumber:  strings.TrimSpace(record[0]),
		FirstName:      strings.TrimSpace(record[1]),
		LastName:       strings.TrimSpace(record[2]),
		Email:          strings.TrimSpace(record[3]),
		Faculty:        strings.TrimSpace(record[4]),
		Department:     strings.TrimSpace(record[5]),
		EnrollmentYear: int32(enrollmentYear),
		ClassLevel:     int16(classLevel),
		AdvisorID:      utils.UUIDToPgtype(uuid.Nil), // Will be assigned later
	}, nil
}

func (s *ImportService) toImportJobResponse(job db.ImportJob) dto.ImportJobResponse {
	var startedAt, completedAt *time.Time
	if job.StartedAt.Valid {
		startedAt = &job.StartedAt.Time
	}
	if job.CompletedAt.Valid {
		completedAt = &job.CompletedAt.Time
	}

	// Parse errors JSON (simplified - just return empty array for now)
	var errors []dto.ImportJobError

	progressPercentage := 0
	if job.TotalRecords > 0 {
		progressPercentage = int((job.ProcessedRecords * 100) / job.TotalRecords)
	}

	status := "pending"
	if job.Status.Valid {
		status = string(job.Status.ImportJobStatus)
	}

	return dto.ImportJobResponse{
		JobID:              utils.PgtypeToUUIDString(job.ID),
		FileName:           job.FileName,
		TotalRecords:       int(job.TotalRecords),
		ProcessedRecords:   int(job.ProcessedRecords),
		SuccessfulRecords:  int(job.SuccessfulRecords),
		FailedRecords:      int(job.FailedRecords),
		ProgressPercentage: progressPercentage,
		Status:             status,
		Errors:             errors,
		CreatedBy:          utils.PgtypeToUUIDString(job.CreatedBy),
		StartedAt:          startedAt,
		CompletedAt:        completedAt,
		CreatedAt:          job.CreatedAt.Time,
	}
}

func (s *ImportService) toImportJobSummary(job db.ImportJob) dto.ImportJobSummary {
	var completedAt *time.Time
	if job.CompletedAt.Valid {
		completedAt = &job.CompletedAt.Time
	}

	status := "pending"
	if job.Status.Valid {
		status = string(job.Status.ImportJobStatus)
	}

	return dto.ImportJobSummary{
		JobID:             utils.PgtypeToUUIDString(job.ID),
		FileName:          job.FileName,
		Status:            status,
		TotalRecords:      int(job.TotalRecords),
		SuccessfulRecords: int(job.SuccessfulRecords),
		FailedRecords:     int(job.FailedRecords),
		CreatedAt:         job.CreatedAt.Time,
		CompletedAt:       completedAt,
	}
}
