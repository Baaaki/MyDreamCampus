package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/student/internal/db"
	"github.com/baaaki/mydreamcampus/student/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/student/internal/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeImportStore struct {
	importJobStore
	created   db.CreateImportJobParams
	progress  []db.UpdateImportJobProgressParams
	started   bool
	completed bool
	failed    bool
}

func (f *fakeImportStore) CreateImportJob(_ context.Context, params db.CreateImportJobParams) (db.ImportJob, error) {
	f.created = params
	return db.ImportJob{ID: utils.UUIDToPgtype(uuid.New())}, nil
}

func (f *fakeImportStore) StartImportJob(context.Context, uuid.UUID) error {
	f.started = true
	return nil
}

func (f *fakeImportStore) UpdateImportJobProgress(_ context.Context, params db.UpdateImportJobProgressParams) error {
	f.progress = append(f.progress, params)
	return nil
}

func (f *fakeImportStore) CompleteImportJob(context.Context, uuid.UUID) error {
	f.completed = true
	return nil
}

func (f *fakeImportStore) FailImportJob(context.Context, uuid.UUID) error {
	f.failed = true
	return nil
}

func (f *fakeImportStore) lastProgress(t *testing.T) db.UpdateImportJobProgressParams {
	t.Helper()
	require.NotEmpty(t, f.progress, "progress was never saved")
	return f.progress[len(f.progress)-1]
}

func (f *fakeImportStore) lastErrors(t *testing.T) []dto.ImportJobError {
	t.Helper()
	var rowErrors []dto.ImportJobError
	require.NoError(t, json.Unmarshal(f.lastProgress(t).Errors, &rowErrors), "errors column must stay valid JSON")
	return rowErrors
}

// fakeCreator plays StudentService.CreateStudent: it rejects a number it has
// seen before, like the real uniqueness check.
type fakeCreator struct {
	calls  []dto.CreateStudentRequest
	seen   map[string]bool
	errFor map[string]error
}

func newFakeCreator() *fakeCreator {
	return &fakeCreator{seen: map[string]bool{}, errFor: map[string]error{}}
}

func (f *fakeCreator) CreateStudent(_ context.Context, req dto.CreateStudentRequest) (dto.StudentResponse, error) {
	f.calls = append(f.calls, req)
	if err, ok := f.errFor[req.StudentNumber]; ok {
		return dto.StudentResponse{}, err
	}
	if f.seen[req.StudentNumber] {
		return dto.StudentResponse{}, serviceErrors.ErrStudentNumberExists
	}
	f.seen[req.StudentNumber] = true
	return dto.StudentResponse{StudentNumber: req.StudentNumber}, nil
}

type fakeImportStaff struct {
	StaffServiceInterface
	instructors map[string][]uuid.UUID
}

func (f *fakeImportStaff) GetInstructorsByDepartment(_ context.Context, department string) ([]uuid.UUID, error) {
	ids, ok := f.instructors[department]
	if !ok {
		return nil, errors.New("staff unavailable")
	}
	return ids, nil
}

func importCSV(lines ...string) string {
	return strings.Join(append([]string{strings.Join(expectedCSVHeader, ",")}, lines...), "\n") + "\n"
}

func csvLine(number, email string) string {
	return number + ",Ali,Veli," + email + ",Mühendislik,Bilgisayar Mühendisliği,2024,1"
}

// runImport parses like the request half of BulkImportStudents and then runs
// processImport synchronously instead of racing its goroutine.
func runImport(t *testing.T, store *fakeImportStore, creator *fakeCreator, staff *fakeImportStaff, csvBody string) {
	t.Helper()
	rows, rowErrors, err := readImportRows(strings.NewReader(csvBody))
	require.NoError(t, err)
	NewImportService(store, creator, staff).processImport(context.Background(), uuid.New(), rows, rowErrors)
}

func TestImportService_ProcessImport_EveryRowGoesThroughCreateStudent(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	advisor := uuid.New()
	store := &fakeImportStore{}
	creator := newFakeCreator()
	staff := &fakeImportStaff{instructors: map[string][]uuid.UUID{"Bilgisayar Mühendisliği": {advisor}}}

	runImport(t, store, creator, staff, importCSV(
		csvLine("2024001", "a@univ.edu"),
		csvLine("2024002", "b@univ.edu"),
		csvLine("2024003", "c@univ.edu"),
	))

	require.Len(t, creator.calls, 3)
	for i, call := range creator.calls {
		assert.Equal(t, fmt.Sprintf("202400%d", i+1), call.StudentNumber)
		require.NotNil(t, call.AdvisorID, "the department's instructor is assigned before create")
		assert.Equal(t, advisor, *call.AdvisorID)
	}
	assert.True(t, store.started)
	assert.True(t, store.completed)
	assert.False(t, store.failed)

	last := store.lastProgress(t)
	assert.Equal(t, int32(3), last.ProcessedRecords)
	assert.Equal(t, int32(3), last.SuccessfulRecords)
	assert.Equal(t, int32(0), last.FailedRecords)
	assert.JSONEq(t, `[]`, string(last.Errors))
}

func TestImportService_ProcessImport_DuplicateRow_ReportedWithStudentNumberExists(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	store := &fakeImportStore{}
	creator := newFakeCreator()

	runImport(t, store, creator, &fakeImportStaff{}, importCSV(
		csvLine("2024001", "a@univ.edu"),
		csvLine("2024001", "other@univ.edu"),
		csvLine("2024002", "geçersiz"),
	))

	// The invalid e-mail row never reaches the create path.
	assert.Len(t, creator.calls, 2)
	assert.True(t, store.completed, "a job with at least one created student completes")
	assert.False(t, store.failed)

	last := store.lastProgress(t)
	assert.Equal(t, int32(3), last.ProcessedRecords)
	assert.Equal(t, int32(1), last.SuccessfulRecords)
	assert.Equal(t, int32(2), last.FailedRecords)

	rowErrors := store.lastErrors(t)
	require.Len(t, rowErrors, 2)
	assert.Equal(t, dto.ImportJobError{
		Row: 4, StudentNumber: "2024002",
		ErrorCode: serviceErrors.ErrInvalidCSVFormat.Code, Message: "geçersiz e-posta: geçersiz",
	}, rowErrors[0])
	assert.Equal(t, dto.ImportJobError{
		Row: 3, StudentNumber: "2024001",
		ErrorCode: serviceErrors.ErrStudentNumberExists.Code, Message: serviceErrors.ErrStudentNumberExists.Message,
	}, rowErrors[1])
}

func TestImportService_ProcessImport_QuotedMessage_KeepsErrorsValidJSON(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	store := &fakeImportStore{}
	creator := newFakeCreator()
	// A Postgres-style message: the old hand-built JSON broke on these quotes.
	quoted := `duplicate key value violates unique constraint "students_email_key"`
	creator.errFor["2024001"] = sharedErrors.New("STUDENT_EMAIL_EXISTS", quoted, http.StatusConflict)
	creator.errFor["2024002"] = fmt.Errorf("query failed: %s", quoted)

	runImport(t, store, creator, &fakeImportStaff{}, importCSV(
		csvLine("2024001", "a@univ.edu"),
		csvLine("2024002", "b@univ.edu"),
		// CSV-quoted cell whose value is "x"@univ.edu.
		csvLine("2024003", `"""x""@univ.edu"`),
	))

	rowErrors := store.lastErrors(t)
	require.Len(t, rowErrors, 3)
	assert.Equal(t, `geçersiz e-posta: "x"@univ.edu`, rowErrors[0].Message)
	assert.Equal(t, quoted, rowErrors[1].Message, "a 4xx keeps the create path's own message")
	// A non-AppError is internal: its text stays in the log, not in the row.
	assert.Equal(t, sharedErrors.ErrInternal.Code, rowErrors[2].ErrorCode)
	assert.Equal(t, "Öğrenci oluşturulamadı", rowErrors[2].Message)
}

func TestImportService_ProcessImport_NoRowSucceeds_FailsJob(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	store := &fakeImportStore{}
	creator := newFakeCreator()
	creator.errFor["2024001"] = serviceErrors.ErrStudentEmailExists

	runImport(t, store, creator, &fakeImportStaff{}, importCSV(
		csvLine("2024001", "a@univ.edu"),
		csvLine("2024002", "yok"),
	))

	assert.True(t, store.failed)
	assert.False(t, store.completed)
	last := store.lastProgress(t)
	assert.Equal(t, int32(2), last.ProcessedRecords)
	assert.Equal(t, int32(0), last.SuccessfulRecords)
	assert.Equal(t, int32(2), last.FailedRecords)
}

func TestImportService_ProcessImport_LargeFile_SavesProgressEvery25Rows(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	store := &fakeImportStore{}
	creator := newFakeCreator()

	lines := make([]string, 0, 60)
	for i := range 60 {
		lines = append(lines, csvLine(fmt.Sprintf("2024%03d", i), fmt.Sprintf("s%d@univ.edu", i)))
	}
	runImport(t, store, creator, &fakeImportStaff{}, importCSV(lines...))

	require.Len(t, store.progress, 3, "after rows 25 and 50, then the final write")
	assert.Equal(t, int32(25), store.progress[0].ProcessedRecords)
	assert.Equal(t, int32(50), store.progress[1].ProcessedRecords)
	assert.Equal(t, int32(60), store.progress[2].ProcessedRecords)
	assert.True(t, store.completed)
}

func TestImportService_BulkImportStudents_BadHeader_ReturnsInvalidCSVFormat(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	store := &fakeImportStore{}
	svc := NewImportService(store, newFakeCreator(), &fakeImportStaff{})

	_, err := svc.BulkImportStudents(context.Background(), "x.csv",
		strings.NewReader("number,name\n1,Ali\n"), uuid.New())

	require.Error(t, err)
	assert.True(t, sharedErrors.Is(err, serviceErrors.ErrInvalidCSVFormat))
	appErr, ok := sharedErrors.As(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, appErr.HTTPStatus)
	assert.Contains(t, appErr.Message, strings.Join(expectedImportHeader, ","))
	assert.Empty(t, store.created.FileName, "no job is recorded for a rejected file")
}

func TestImportService_ToImportJobResponse_ReadsRowErrorsBack(t *testing.T) {
	svc := NewImportService(&fakeImportStore{}, newFakeCreator(), &fakeImportStaff{})
	stored := []dto.ImportJobError{{Row: 2, StudentNumber: "1", ErrorCode: "X", Message: `"q"`}}
	raw, err := json.Marshal(stored)
	require.NoError(t, err)

	resp := svc.toImportJobResponse(db.ImportJob{Errors: raw})
	assert.Equal(t, stored, resp.Errors)

	// Jobs from before the structured format hold plain strings.
	legacy := svc.toImportJobResponse(db.ImportJob{Errors: []byte(`["Line 2: bad"]`)})
	assert.Empty(t, legacy.Errors)
}
