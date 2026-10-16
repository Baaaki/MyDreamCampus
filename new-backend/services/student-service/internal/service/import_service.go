package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/student/internal/db"
	"github.com/baaaki/mydreamcampus/student/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/student/internal/errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// importJobStore is the part of the import repository the service uses; an
// interface so the background run can be tested without a database.
type importJobStore interface {
	CreateImportJob(ctx context.Context, params db.CreateImportJobParams) (db.ImportJob, error)
	GetImportJobByID(ctx context.Context, id uuid.UUID) (db.ImportJob, error)
	UpdateImportJobProgress(ctx context.Context, params db.UpdateImportJobProgressParams) error
	StartImportJob(ctx context.Context, id uuid.UUID) error
	CompleteImportJob(ctx context.Context, id uuid.UUID) error
	FailImportJob(ctx context.Context, id uuid.UUID) error
	ListImportJobsByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]db.ImportJob, int64, error)
}

// StudentCreator is the single-student create path. The import goes through
// it row by row: a direct insert skips the student.created outbox event, and
// without it auth never opens the login account and attendance, grades and
// meal never learn the student exists.
type StudentCreator interface {
	CreateStudent(ctx context.Context, req dto.CreateStudentRequest) (dto.StudentResponse, error)
}

// importProgressEvery is how many rows run between progress writes, so the
// admin's poll sees the job advance on a large file.
const importProgressEvery = 25

type ImportService struct {
	importRepo   importJobStore
	creator      StudentCreator
	staffService StaffServiceInterface
}

func NewImportService(
	importRepo importJobStore,
	creator StudentCreator,
	staffService StaffServiceInterface,
) *ImportService {
	return &ImportService{
		importRepo:   importRepo,
		creator:      creator,
		staffService: staffService,
	}
}

// importRow is a parsed CSV row and its line in the file (header is line 1).
type importRow struct {
	line int
	req  dto.CreateStudentRequest
}

var expectedImportHeader = []string{"student_number", "first_name", "last_name", "email", "faculty", "department", "enrollment_year", "class_level"}

// BulkImportStudents parses the CSV, records the job and creates the students
// in the background.
func (s *ImportService) BulkImportStudents(ctx context.Context, fileName string, fileReader io.Reader, createdBy uuid.UUID) (string, error) {
	log := logger.WithContextAndFields(ctx,
		zap.String("service", "ImportService"),
		zap.String("method", "BulkImportStudents"),
	)

	rows, rowErrors, err := readImportRows(fileReader)
	if err != nil {
		log.Warn("rejected import file", zap.Error(err))
		return "", err
	}

	totalRecords := len(rows) + len(rowErrors)
	if totalRecords == 0 {
		return "", errors.WrapWithMessage(serviceErrors.ErrInvalidCSVFormat, nil, "CSV dosyasında öğrenci satırı yok")
	}

	job, err := s.importRepo.CreateImportJob(ctx, db.CreateImportJobParams{
		FileName:     fileName,
		TotalRecords: utils.ClampToInt32(totalRecords),
		CreatedBy:    utils.UUIDToPgtype(createdBy),
	})
	if err != nil {
		log.Error("failed to create import job", zap.Error(err))
		return "", errors.ErrInternal
	}

	jobID := utils.PgtypeToUUIDString(job.ID)

	// The request context ends with the 202 response; the run must not.
	go s.processImport(context.WithoutCancel(ctx), utils.PgtypeToUUID(job.ID), rows, rowErrors)

	log.Info("bulk import job created",
		zap.String("job_id", jobID),
		zap.Int("total_records", totalRecords),
		zap.Int("valid_records", len(rows)),
	)

	return jobID, nil
}

// readImportRows checks the header and parses every row. A row that cannot
// be used becomes a row error so the rest of the file still imports; only a
// file that cannot be read at all is rejected.
func readImportRows(fileReader io.Reader) ([]importRow, []dto.ImportJobError, error) {
	csvReader := csv.NewReader(fileReader)
	csvReader.TrimLeadingSpace = true
	// A row with the wrong column count is reported for that row instead of
	// failing the reader.
	csvReader.FieldsPerRecord = -1

	header, err := csvReader.Read()
	if err != nil {
		return nil, nil, errors.WrapWithMessage(serviceErrors.ErrInvalidCSVFormat, err, "CSV dosyası okunamadı")
	}
	if !validateHeader(header, expectedImportHeader) {
		return nil, nil, errors.WrapWithMessage(serviceErrors.ErrInvalidCSVFormat,
			fmt.Errorf("unexpected CSV header %q", header),
			"CSV başlığı şu sütunlar olmalı: "+strings.Join(expectedImportHeader, ","))
	}

	var rows []importRow
	var rowErrors []dto.ImportJobError
	for line := 2; ; line++ {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			var parseErr *csv.ParseError
			if !stderrors.As(err, &parseErr) {
				return nil, nil, errors.WrapWithMessage(serviceErrors.ErrInvalidCSVFormat, err, "CSV dosyası okunamadı")
			}
			rowErrors = append(rowErrors, dto.ImportJobError{
				Row: line, ErrorCode: serviceErrors.ErrInvalidCSVFormat.Code, Message: "Satır okunamadı",
			})
			continue
		}

		req, parseErr := parseStudentRecord(record)
		if parseErr != nil {
			rowErrors = append(rowErrors, dto.ImportJobError{
				Row: line, StudentNumber: firstCell(record), ErrorCode: serviceErrors.ErrInvalidCSVFormat.Code, Message: parseErr.Error(),
			})
			continue
		}
		rows = append(rows, importRow{line: line, req: req})
	}
	return rows, rowErrors, nil
}

// processImport creates each valid row through the single-student path and
// keeps the job's counters and row errors current.
func (s *ImportService) processImport(ctx context.Context, jobID uuid.UUID, rows []importRow, rowErrors []dto.ImportJobError) {
	log := logger.WithContextAndFields(ctx,
		zap.String("service", "ImportService"),
		zap.String("method", "processImport"),
		zap.String("job_id", jobID.String()),
	)

	if err := s.importRepo.StartImportJob(ctx, jobID); err != nil {
		log.Error("failed to start import job", zap.Error(err))
		return
	}

	s.autoAssignAdvisors(ctx, rows)

	failed := rowErrors
	succeeded := 0
	for i, row := range rows {
		if _, err := s.creator.CreateStudent(ctx, row.req); err != nil {
			log.Warn("import row rejected",
				zap.Int("line", row.line),
				zap.String("student_number", row.req.StudentNumber),
				zap.Error(err),
			)
			failed = append(failed, createRowError(row, err))
		} else {
			succeeded++
		}
		if (i+1)%importProgressEvery == 0 {
			s.saveProgress(ctx, jobID, succeeded, failed)
		}
	}
	s.saveProgress(ctx, jobID, succeeded, failed)

	if succeeded == 0 && len(failed) > 0 {
		if err := s.importRepo.FailImportJob(ctx, jobID); err != nil {
			log.Error("failed to mark import job as failed", zap.Error(err))
		}
	} else if err := s.importRepo.CompleteImportJob(ctx, jobID); err != nil {
		log.Error("failed to mark import job as completed", zap.Error(err))
	}

	log.Info("import processing completed",
		zap.Int("success", succeeded),
		zap.Int("failed", len(failed)),
	)
}

func (s *ImportService) saveProgress(ctx context.Context, jobID uuid.UUID, succeeded int, failed []dto.ImportJobError) {
	if failed == nil {
		failed = []dto.ImportJobError{}
	}
	errorsJSON, err := json.Marshal(failed)
	if err != nil {
		logger.Error("failed to encode import errors", zap.Error(err), zap.String("job_id", jobID.String()))
		errorsJSON = []byte("[]")
	}
	if err := s.importRepo.UpdateImportJobProgress(ctx, db.UpdateImportJobProgressParams{
		ID:                utils.UUIDToPgtype(jobID),
		ProcessedRecords:  utils.ClampToInt32(succeeded + len(failed)),
		SuccessfulRecords: utils.ClampToInt32(succeeded),
		FailedRecords:     utils.ClampToInt32(len(failed)),
		Errors:            errorsJSON,
	}); err != nil {
		logger.Error("failed to update import job progress", zap.Error(err), zap.String("job_id", jobID.String()))
	}
}

// createRowError reports a rejected row with the create path's own message
// when it is the admin's to fix (duplicate number or e-mail); anything else
// is an internal failure the admin cannot act on.
func createRowError(row importRow, err error) dto.ImportJobError {
	rowErr := dto.ImportJobError{
		Row:           row.line,
		StudentNumber: row.req.StudentNumber,
		ErrorCode:     errors.ErrInternal.Code,
		Message:       "Öğrenci oluşturulamadı",
	}
	if appErr, ok := errors.As(err); ok && appErr.HTTPStatus < http.StatusInternalServerError {
		rowErr.ErrorCode = appErr.Code
		rowErr.Message = appErr.Message
	}
	return rowErr
}

func firstCell(record []string) string {
	if len(record) == 0 {
		return ""
	}
	return strings.TrimSpace(record[0])
}

// autoAssignAdvisors gives each row an advisor from its department's
// instructors, round-robin. A department staff cannot answer for keeps its
// students without an advisor rather than failing the import.
func (s *ImportService) autoAssignAdvisors(ctx context.Context, rows []importRow) {
	log := logger.WithContextAndFields(ctx,
		zap.String("service", "ImportService"),
		zap.String("method", "autoAssignAdvisors"),
	)

	departmentRows := make(map[string][]int)
	for i, row := range rows {
		departmentRows[row.req.Department] = append(departmentRows[row.req.Department], i)
	}

	for department, indices := range departmentRows {
		instructors, err := s.staffService.GetInstructorsByDepartment(ctx, department)
		if err != nil {
			log.Warn("failed to get instructors for department",
				zap.String("department", department),
				zap.Error(err),
			)
			continue
		}
		if len(instructors) == 0 {
			log.Warn("no instructors found for department",
				zap.String("department", department),
			)
			continue
		}

		for i, rowIdx := range indices {
			advisorID := instructors[i%len(instructors)]
			rows[rowIdx].req.AdvisorID = &advisorID
		}

		log.Info("assigned advisors for department",
			zap.String("department", department),
			zap.Int("student_count", len(indices)),
			zap.Int("instructor_count", len(instructors)),
		)
	}
}

// GetImportJobStatus retrieves import job status
func (s *ImportService) GetImportJobStatus(ctx context.Context, jobID string) (dto.ImportJobResponse, error) {
	log := logger.WithContextAndFields(ctx,
		zap.String("service", "ImportService"),
		zap.String("method", "GetImportJobStatus"),
	)

	id, err := uuid.Parse(jobID)
	if err != nil {
		return dto.ImportJobResponse{}, errors.ErrInvalidID
	}

	job, err := s.importRepo.GetImportJobByID(ctx, id)
	if err != nil {
		log.Error("failed to get import job",
			zap.Error(err),
			zap.String("job_id", jobID),
		)
		return dto.ImportJobResponse{}, errors.ErrNotFound
	}

	return s.toImportJobResponse(job), nil
}

// ListImportJobs lists import jobs for a user
func (s *ImportService) ListImportJobs(ctx context.Context, userID uuid.UUID, query dto.ImportJobFilterQuery) (dto.ImportJobListResponse, error) {
	log := logger.WithContextAndFields(ctx,
		zap.String("service", "ImportService"),
		zap.String("method", "ListImportJobs"),
	)

	// The handler's binding already guarantees both are >= 1; a direct
	// caller with a zero query must not divide by zero below.
	page := max(query.Page, 1)
	limit := query.Limit
	if limit < 1 {
		limit = 20
	}

	jobs, total, err := s.importRepo.ListImportJobsByUser(ctx, userID,
		utils.ClampToInt32(limit), utils.ClampToInt32((page-1)*limit))
	if err != nil {
		log.Error("failed to list import jobs",
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
			Page:       page,
			Limit:      limit,
			Total:      int(total),
			TotalPages: (int(total) + limit - 1) / limit,
		},
	}, nil
}

// Helper functions

// sanitizeCSVCell strips formula injection characters from CSV cells
func sanitizeCSVCell(cell string) string {
	cell = strings.TrimSpace(cell)
	// Prevent CSV formula injection
	if len(cell) > 0 {
		firstChar := cell[0]
		if firstChar == '=' || firstChar == '+' || firstChar == '-' || firstChar == '@' || firstChar == '\t' || firstChar == '\r' {
			cell = "'" + cell // Prefix with single quote to neutralize formula
		}
	}
	return cell
}

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

// parseStudentRecord turns a CSV row into a create request. Its error is the
// message the admin sees next to the row, so it is in Turkish.
func parseStudentRecord(record []string) (dto.CreateStudentRequest, error) {
	if len(record) != len(expectedImportHeader) {
		return dto.CreateStudentRequest{}, fmt.Errorf("%d sütun bekleniyordu, %d sütun var", len(expectedImportHeader), len(record))
	}

	for i := range record {
		record[i] = sanitizeCSVCell(record[i])
	}

	required := []struct{ value, label string }{
		{record[0], "öğrenci numarası"},
		{record[1], "ad"},
		{record[2], "soyad"},
		{record[3], "e-posta"},
		{record[4], "fakülte"},
		{record[5], "bölüm"},
	}
	for _, field := range required {
		if field.value == "" {
			return dto.CreateStudentRequest{}, fmt.Errorf("%s boş", field.label)
		}
	}

	if addr, err := mail.ParseAddress(record[3]); err != nil || addr.Address != record[3] {
		return dto.CreateStudentRequest{}, fmt.Errorf("geçersiz e-posta: %s", record[3])
	}

	enrollmentYear, err := strconv.Atoi(record[6])
	if err != nil || enrollmentYear < 1900 || enrollmentYear > 2100 {
		return dto.CreateStudentRequest{}, fmt.Errorf("geçersiz giriş yılı: %s", record[6])
	}

	classLevel, err := strconv.ParseInt(record[7], 10, 16)
	if err != nil {
		return dto.CreateStudentRequest{}, fmt.Errorf("geçersiz sınıf: %s", record[7])
	}
	if classLevel < 1 || classLevel > 6 {
		return dto.CreateStudentRequest{}, fmt.Errorf("sınıf 1 ile 6 arasında olmalı: %s", record[7])
	}

	return dto.CreateStudentRequest{
		StudentNumber:  record[0],
		FirstName:      record[1],
		LastName:       record[2],
		Email:          record[3],
		Faculty:        record[4],
		Department:     record[5],
		EnrollmentYear: enrollmentYear,
		ClassLevel:     int16(classLevel),
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

	// Jobs written before row errors were structured hold plain strings;
	// they show their counters without row details.
	var rowErrors []dto.ImportJobError
	if len(job.Errors) > 0 {
		if err := json.Unmarshal(job.Errors, &rowErrors); err != nil {
			rowErrors = nil
		}
	}

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
		Errors:             rowErrors,
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
