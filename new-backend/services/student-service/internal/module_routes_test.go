package student

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/student/internal/db"
	"github.com/baaaki/mydreamcampus/student/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/student/internal/errors"
	"github.com/baaaki/mydreamcampus/student/internal/handler"
	"github.com/baaaki/mydreamcampus/student/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const routesInternalSecret = "test-internal-secret-minimum-32-characters"

// fakeStudentStore answers like repository.StudentRepository: a missing or
// deactivated record is ErrStudentNotFoundRepo, an unknown number or e-mail
// is an empty row, and create fills the event's id in.
type fakeStudentStore struct {
	service.StudentRepositoryInterface
	rows   map[uuid.UUID]db.Student
	events []map[string]any
}

func (f *fakeStudentStore) find(match func(db.Student) bool) db.Student {
	for _, s := range f.rows {
		if match(s) {
			return s
		}
	}
	return db.Student{}
}

func (f *fakeStudentStore) GetStudentByNumber(_ context.Context, n string) (db.Student, error) {
	return f.find(func(s db.Student) bool { return s.StudentNumber == n }), nil
}

func (f *fakeStudentStore) GetStudentByEmail(_ context.Context, e string) (db.Student, error) {
	return f.find(func(s db.Student) bool { return s.Email == e }), nil
}

func (f *fakeStudentStore) active(id uuid.UUID) (db.Student, error) {
	s, ok := f.rows[id]
	if !ok || !s.IsActive.Bool {
		return db.Student{}, fmt.Errorf("%w: student with id %s not found", serviceErrors.ErrStudentNotFoundRepo, id)
	}
	return s, nil
}

func (f *fakeStudentStore) GetStudentByID(_ context.Context, id uuid.UUID) (db.Student, error) {
	return f.active(id)
}

func (f *fakeStudentStore) CreateStudentWithEvent(_ context.Context, p db.CreateStudentParams, event map[string]any) (db.Student, error) {
	id := uuid.New()
	s := db.Student{
		ID: utils.UUIDToPgtype(id), StudentNumber: p.StudentNumber, FirstName: p.FirstName, LastName: p.LastName,
		Email: p.Email, Faculty: p.Faculty, Department: p.Department, EnrollmentYear: p.EnrollmentYear,
		ClassLevel: p.ClassLevel, AdvisorID: p.AdvisorID, AdvisorName: p.AdvisorName,
		Status: pgtype.Text{String: "active", Valid: true}, IsActive: pgtype.Bool{Bool: true, Valid: true},
	}
	f.rows[id] = s
	event["id"] = id.String()
	f.events = append(f.events, event)
	return s, nil
}

func (f *fakeStudentStore) UpdateStudentWithEvent(_ context.Context, id uuid.UUID, p db.UpdateStudentParams, event map[string]any) (db.Student, error) {
	s, err := f.active(id)
	if err != nil {
		return s, err
	}
	if p.FirstName.Valid {
		s.FirstName = p.FirstName.String
	}
	if p.Email.Valid {
		s.Email = p.Email.String
	}
	f.rows[id] = s
	f.events = append(f.events, event)
	return s, nil
}

func (f *fakeStudentStore) SoftDeleteStudentWithEvent(_ context.Context, id uuid.UUID, event map[string]any) error {
	s, err := f.active(id)
	if err != nil {
		return err
	}
	s.IsActive = pgtype.Bool{Bool: false, Valid: true}
	f.rows[id] = s
	f.events = append(f.events, event)
	return nil
}

func (f *fakeStudentStore) ListStudentsByAdvisor(_ context.Context, advisorID uuid.UUID) ([]db.Student, error) {
	var out []db.Student
	for _, s := range f.rows {
		if s.IsActive.Bool && utils.PgtypeToUUID(s.AdvisorID) == advisorID {
			out = append(out, s)
		}
	}
	return out, nil
}

type fakeStaffClient struct{}

func (fakeStaffClient) ValidateAdvisor(context.Context, uuid.UUID) error { return nil }

func (fakeStaffClient) GetAdvisorInfo(_ context.Context, id uuid.UUID) (*service.AdvisorDetails, error) {
	return &service.AdvisorDetails{ID: id.String(), Name: "Ahmet Yılmaz"}, nil
}

func (fakeStaffClient) GetInstructorsByDepartment(context.Context, string) ([]uuid.UUID, error) {
	return nil, nil
}

// The import job routes are covered by their own handler tests; here they
// only need to exist behind the admin guard.
type fakeImportService struct{}

func (fakeImportService) BulkImportStudents(context.Context, string, io.Reader, uuid.UUID) (string, error) {
	return uuid.NewString(), nil
}

func (fakeImportService) GetImportJobStatus(_ context.Context, id string) (dto.ImportJobResponse, error) {
	return dto.ImportJobResponse{JobID: id}, nil
}

func (fakeImportService) ListImportJobs(context.Context, uuid.UUID, dto.ImportJobFilterQuery) (dto.ImportJobListResponse, error) {
	return dto.ImportJobListResponse{}, nil
}

type studentRoutes struct {
	router    *gin.Engine
	store     *fakeStudentStore
	advisorID uuid.UUID
	adviseeID uuid.UUID
	otherID   uuid.UUID
	demoID    uuid.UUID
}

// newStudentRoutes mounts the module's own RegisterRoutes and
// RegisterPublicRoutes over a fake store: two students advised by one
// teacher, one by nobody, and the protected demo student.
func newStudentRoutes(t *testing.T) *studentRoutes {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	utils.InitJWTSecret("test-secret-key-minimum-32-characters-long-aaa")
	platformMiddleware.SetBlacklistChecker(nil)

	cfg := &config.Config{}
	cfg.Server.InternalSecret = routesInternalSecret
	cfg.ProtectedAccountEmails = []string{"zeynep.sahin@uni.edu.tr"}

	s := &studentRoutes{
		store:     &fakeStudentStore{rows: map[uuid.UUID]db.Student{}},
		advisorID: uuid.New(), adviseeID: uuid.New(), otherID: uuid.New(), demoID: uuid.New(),
	}
	seed := func(id uuid.UUID, number, email string, advisor *uuid.UUID) {
		row := db.Student{
			ID: utils.UUIDToPgtype(id), StudentNumber: number, FirstName: "Ad", LastName: "Soyad", Email: email,
			Faculty: "Mühendislik", Department: "Bilgisayar Mühendisliği", EnrollmentYear: 2024, ClassLevel: 2,
			Status: pgtype.Text{String: "active", Valid: true}, IsActive: pgtype.Bool{Bool: true, Valid: true},
		}
		if advisor != nil {
			row.AdvisorID = utils.UUIDToPgtype(*advisor)
		}
		s.store.rows[id] = row
	}
	seed(s.adviseeID, "2024001", "ali@uni.edu.tr", &s.advisorID)
	seed(s.otherID, "2024002", "ayse@uni.edu.tr", nil)
	seed(s.demoID, "2024003", "zeynep.sahin@uni.edu.tr", &s.advisorID)

	m := &Module{
		cfg:            cfg,
		studentHandler: handler.NewStudentHandler(service.NewStudentService(s.store, fakeStaffClient{}, cfg), fakeImportService{}),
	}
	s.router = gin.New()
	m.RegisterRoutes(s.router.Group("/api/students"))
	m.RegisterPublicRoutes(s.router)
	return s
}

func bearer(t *testing.T, userID uuid.UUID, role string) string {
	t.Helper()
	tok, _, err := utils.GenerateAccessToken(userID.String(), role, "", 0)
	require.NoError(t, err)
	return "Bearer " + tok
}

func (s *studentRoutes) do(method, path, auth, body string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

// A student record carries contact details, so a single read is narrowed to
// the student themself and their advisor.
func TestStudentRoutes_SingleRead_NarrowedToOwnerAndAdvisor(t *testing.T) {
	s := newStudentRoutes(t)
	advisee := "/api/students/" + s.adviseeID.String()
	other := "/api/students/" + s.otherID.String()

	cases := []struct {
		name, path, auth string
		want             int
	}{
		{"student reads own record", advisee, bearer(t, s.adviseeID, "student"), http.StatusOK},
		{"student reads another student", other, bearer(t, s.adviseeID, "student"), http.StatusForbidden},
		{"student probes a missing id", "/api/students/" + uuid.NewString(), bearer(t, s.adviseeID, "student"), http.StatusForbidden},
		{"advisor reads an advisee", advisee, bearer(t, s.advisorID, "teacher"), http.StatusOK},
		{"teacher reads someone else's advisee", advisee, bearer(t, uuid.New(), "teacher"), http.StatusForbidden},
		{"teacher reads a student with no advisor", other, bearer(t, s.advisorID, "teacher"), http.StatusForbidden},
		{"admin reads anyone", other, bearer(t, uuid.New(), "admin"), http.StatusOK},
		{"unknown role", advisee, bearer(t, uuid.New(), "guest"), http.StatusForbidden},
		{"no token", advisee, "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := s.do(http.MethodGet, tc.path, tc.auth, "")
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}

func TestStudentRoutes_AdminOnlyRoutes_RejectOthers(t *testing.T) {
	s := newStudentRoutes(t)
	teacher := bearer(t, s.advisorID, "teacher")
	student := bearer(t, s.adviseeID, "student")
	one := "/api/students/" + s.otherID.String()

	cases := []struct{ name, method, path, auth string }{
		{"list as teacher", http.MethodGet, "/api/students", teacher},
		{"search as teacher", http.MethodPost, "/api/students/search", teacher},
		{"create as teacher", http.MethodPost, "/api/students", teacher},
		{"update as student", http.MethodPut, one, student},
		{"delete as teacher", http.MethodDelete, one, teacher},
		{"orphaned as teacher", http.MethodGet, "/api/students/orphaned", teacher},
		{"bulk advisor assign as teacher", http.MethodPut, "/api/students/bulk-advisor-assign", teacher},
		{"bulk import as teacher", http.MethodPost, "/api/students/bulk-import", teacher},
		{"import jobs as student", http.MethodGet, "/api/students/bulk-import", student},
		{"advisees as student", http.MethodGet, "/api/students/my-advisees", student},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := s.do(tc.method, tc.path, tc.auth, `{}`)
			assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		})
	}
	assert.Empty(t, s.store.events, "a refused request must not reach the outbox")
}

func TestStudentRoutes_InternalRoutes_RequireSecret(t *testing.T) {
	s := newStudentRoutes(t)
	path := "/internal/students/" + s.adviseeID.String()

	assert.Equal(t, http.StatusUnauthorized, s.do(http.MethodGet, path, "", "").Code)
	assert.Equal(t, http.StatusUnauthorized, s.do(http.MethodGet, path, "", "", "X-Internal-Secret", "wrong").Code)

	w := s.do(http.MethodGet, path, "", "", "X-Internal-Secret", routesInternalSecret)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var got dto.StudentResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "ali@uni.edu.tr", got.Email)
}

func TestStudentRoutes_AdvisorSeesOnlyOwnAdvisees(t *testing.T) {
	s := newStudentRoutes(t)

	w := s.do(http.MethodGet, "/api/students/my-advisees", bearer(t, s.advisorID, "teacher"), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := w.Body.String()
	assert.Contains(t, body, s.adviseeID.String())
	assert.Contains(t, body, s.demoID.String())
	assert.NotContains(t, body, s.otherID.String())
}

func TestStudentRoutes_AdminLifecycle_WritesThroughOutbox(t *testing.T) {
	s := newStudentRoutes(t)
	admin := bearer(t, uuid.New(), "admin")
	body := fmt.Sprintf(`{"student_number":"2025001","first_name":"Can","last_name":"Öz","email":"can@uni.edu.tr",
		"faculty":"Mühendislik","department":"Fizik","enrollment_year":2025,"class_level":1,"advisor_id":%q}`, s.advisorID)

	w := s.do(http.MethodPost, "/api/students", admin, body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created dto.StudentResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "can@uni.edu.tr", created.Email)
	require.NotNil(t, created.AdvisorID)
	assert.Equal(t, s.advisorID.String(), *created.AdvisorID)
	require.Len(t, s.store.events, 1)
	assert.Equal(t, created.ID, s.store.events[0]["id"], "student.created carries the new row's id")

	assert.Equal(t, http.StatusConflict, s.do(http.MethodPost, "/api/students", admin, body).Code, "same number twice")

	w = s.do(http.MethodPut, "/api/students/"+created.ID, admin, `{"first_name":"Cem"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"first_name":"Cem"`)

	w = s.do(http.MethodDelete, "/api/students/"+created.ID, admin, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, s.store.events, 3)
	assert.Equal(t, created.ID, s.store.events[2]["id"], "student.deactivated")
	assert.Equal(t, false, s.store.events[2]["is_active"])

	assert.Equal(t, http.StatusNotFound, s.do(http.MethodGet, "/api/students/"+created.ID, admin, "").Code)
	assert.Equal(t, http.StatusNotFound, s.do(http.MethodDelete, "/api/students/"+created.ID, admin, "").Code)
}

// The demo student's login is printed on the login page: deleting or
// deactivating it would lock every visitor out until the next rewind.
func TestStudentRoutes_ProtectedAccount_CannotBeDeletedOrDeactivated(t *testing.T) {
	s := newStudentRoutes(t)
	admin := bearer(t, uuid.New(), "admin")
	demo := "/api/students/" + s.demoID.String()

	assert.Equal(t, http.StatusForbidden, s.do(http.MethodDelete, demo, admin, "").Code)
	assert.Equal(t, http.StatusForbidden, s.do(http.MethodPut, demo, admin, `{"status":"suspended"}`).Code)
	assert.Equal(t, http.StatusForbidden, s.do(http.MethodPut, demo, admin, `{"email":"someone@uni.edu.tr"}`).Code)
	assert.Empty(t, s.store.events)
	assert.True(t, s.store.rows[s.demoID].IsActive.Bool)
}

func TestStudentRoutes_BadInput_Returns400(t *testing.T) {
	s := newStudentRoutes(t)
	admin := bearer(t, uuid.New(), "admin")

	cases := []struct{ name, method, path, body string }{
		{"create without a number", http.MethodPost, "/api/students", `{"first_name":"A","last_name":"B","email":"a@uni.edu.tr","faculty":"F","department":"D","enrollment_year":2025,"class_level":1}`},
		{"create with class level 9", http.MethodPost, "/api/students", `{"student_number":"1","first_name":"A","last_name":"B","email":"a@uni.edu.tr","faculty":"F","department":"D","enrollment_year":2025,"class_level":9}`},
		{"create with a bad e-mail", http.MethodPost, "/api/students", `{"student_number":"1","first_name":"A","last_name":"B","email":"nope","faculty":"F","department":"D","enrollment_year":2025,"class_level":1}`},
		{"update with a bad id", http.MethodPut, "/api/students/not-a-uuid", `{"first_name":"X"}`},
		{"delete with a bad id", http.MethodDelete, "/api/students/not-a-uuid", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := s.do(tc.method, tc.path, admin, tc.body)
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}
	assert.Empty(t, s.store.events)
}
