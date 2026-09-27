package staff

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/config"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	platformMiddleware "github.com/baaaki/mydreamcampus/shared/platform/middleware"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/staff/internal/db"
	"github.com/baaaki/mydreamcampus/staff/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/staff/internal/errors"
	"github.com/baaaki/mydreamcampus/staff/internal/handler"
	"github.com/baaaki/mydreamcampus/staff/internal/repository"
	"github.com/baaaki/mydreamcampus/staff/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const routesInternalSecret = "test-internal-secret-minimum-32-characters"

// fakeStaffStore answers like repository.StaffRepository: a missing or
// deactivated record is ErrStaffNotFoundRepo, and an unknown e-mail is an
// empty row rather than an error.
type fakeStaffStore struct {
	rows   map[uuid.UUID]db.Staff
	events []map[string]any
}

func (f *fakeStaffStore) GetStaffByEmail(_ context.Context, email string) (db.Staff, error) {
	for _, s := range f.rows {
		if s.Email == email {
			return s, nil
		}
	}
	return db.Staff{}, nil
}

func (f *fakeStaffStore) CreateStaffWithEvent(_ context.Context, p db.CreateStaffParams, event map[string]any) (db.Staff, error) {
	id := uuid.New()
	s := db.Staff{
		ID: utils.UUIDToPgtype(id), Email: p.Email, FirstName: p.FirstName, LastName: p.LastName,
		Role: p.Role, Department: p.Department, Phone: p.Phone, OfficeLocation: p.OfficeLocation, IsActive: true,
	}
	f.rows[id] = s
	// The repository fills the id in inside the create transaction.
	event["id"] = id.String()
	f.events = append(f.events, event)
	return s, nil
}

func (f *fakeStaffStore) active(id uuid.UUID, op string) (db.Staff, error) {
	s, ok := f.rows[id]
	if !ok || !s.IsActive {
		return db.Staff{}, fmt.Errorf("%w: staff with id %s not found for %s", serviceErrors.ErrStaffNotFoundRepo, id, op)
	}
	return s, nil
}

func (f *fakeStaffStore) GetStaffByID(_ context.Context, id uuid.UUID) (db.Staff, error) {
	return f.active(id, "read")
}

func (f *fakeStaffStore) UpdateStaffWithEvent(_ context.Context, id uuid.UUID, p db.UpdateStaffParams, event map[string]any) (db.Staff, error) {
	s, err := f.active(id, "update")
	if err != nil {
		return s, err
	}
	if p.Department.Valid {
		s.Department = p.Department
	}
	if p.OfficeLocation.Valid {
		s.OfficeLocation = p.OfficeLocation
	}
	f.rows[id] = s
	f.events = append(f.events, event)
	return s, nil
}

func (f *fakeStaffStore) SoftDeleteStaffWithEvent(_ context.Context, id uuid.UUID, event map[string]any) error {
	s, err := f.active(id, "deletion")
	if err != nil {
		return err
	}
	s.IsActive = false
	f.rows[id] = s
	f.events = append(f.events, event)
	return nil
}

func (f *fakeStaffStore) ListStaff(_ context.Context, _, _ int32) ([]db.Staff, int64, error) {
	var out []db.Staff
	for _, s := range f.rows {
		if s.IsActive {
			out = append(out, s)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeStaffStore) GetInstructorsByDepartment(_ context.Context, department string) ([]db.Staff, error) {
	var out []db.Staff
	for _, s := range f.rows {
		if s.IsActive && s.Department.String == department {
			out = append(out, s)
		}
	}
	return out, nil
}

type fakeAdminStaffStore struct {
	rows []db.AdminStaff
}

func (f *fakeAdminStaffStore) List(_ context.Context, _ string) ([]db.AdminStaff, error) {
	return f.rows, nil
}

func (f *fakeAdminStaffStore) GetByID(_ context.Context, id uuid.UUID) (db.AdminStaff, error) {
	for _, r := range f.rows {
		if utils.PgtypeToUUID(r.ID) == id {
			return r, nil
		}
	}
	return db.AdminStaff{}, fmt.Errorf("%w: admin staff %s", sharedErrors.ErrNotFoundRepo, id)
}

func (f *fakeAdminStaffStore) Create(_ context.Context, p db.CreateAdminStaffParams) (db.AdminStaff, error) {
	r := db.AdminStaff{ID: utils.UUIDToPgtype(uuid.New()), Email: p.Email, FirstName: p.FirstName, LastName: p.LastName, Faculty: p.Faculty, Position: p.Position}
	f.rows = append(f.rows, r)
	return r, nil
}

func (f *fakeAdminStaffStore) Update(_ context.Context, p db.UpdateAdminStaffParams) (db.AdminStaff, error) {
	return db.AdminStaff{ID: p.ID, Email: p.Email, FirstName: p.FirstName, LastName: p.LastName}, nil
}

type staffRoutes struct {
	router     *gin.Engine
	store      *fakeStaffStore
	adminStore *fakeAdminStaffStore
	teacherID  uuid.UUID
	otherID    uuid.UUID
}

// newStaffRoutes mounts the module's own RegisterRoutes and
// RegisterPublicRoutes over fake stores, so every guard and path is the one
// production serves. The teacher profile repository has no pool: only the
// paths that fail before reaching it are exercised here.
func newStaffRoutes(t *testing.T) *staffRoutes {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	utils.InitJWTSecret("test-secret-key-minimum-32-characters-long-aaa")
	platformMiddleware.SetBlacklistChecker(nil)

	cfg := &config.Config{}
	cfg.Server.InternalSecret = routesInternalSecret

	store := &fakeStaffStore{rows: map[uuid.UUID]db.Staff{}}
	adminStore := &fakeAdminStaffStore{}
	teacherID, otherID := uuid.New(), uuid.New()
	for id, email := range map[uuid.UUID]string{teacherID: "ahmet@uni.edu.tr", otherID: "zeynep.hoca@uni.edu.tr"} {
		store.rows[id] = db.Staff{
			ID: utils.UUIDToPgtype(id), Email: email, FirstName: "Ad", LastName: "Soyad", Role: "teacher",
			Department: pgtype.Text{String: "Bilgisayar Mühendisliği", Valid: true}, IsActive: true,
		}
	}

	staffSvc := service.NewStaffService(store, cfg)
	m := &Module{
		cfg:                   cfg,
		staffHandler:          handler.NewStaffHandler(staffSvc),
		teacherProfileHandler: handler.NewTeacherProfileHandler(service.NewTeacherProfileService(repository.NewTeacherProfileRepository(nil))),
		adminStaffHandler:     handler.NewAdminStaffHandler(service.NewAdminStaffService(adminStore, cfg)),
	}
	r := gin.New()
	m.RegisterRoutes(r.Group("/api/staff"))
	m.RegisterPublicRoutes(r)
	return &staffRoutes{router: r, store: store, adminStore: adminStore, teacherID: teacherID, otherID: otherID}
}

func bearer(t *testing.T, userID, role string) string {
	t.Helper()
	tok, _, err := utils.GenerateAccessToken(userID, role, "", 0)
	require.NoError(t, err)
	return "Bearer " + tok
}

func (s *staffRoutes) do(method, path, auth, body string, header ...string) *httptest.ResponseRecorder {
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

func TestStaffRoutes_Guards_RejectWrongCaller(t *testing.T) {
	s := newStaffRoutes(t)
	teacher := bearer(t, s.teacherID.String(), "teacher")
	student := bearer(t, uuid.NewString(), "student")
	other := "/api/staff/" + s.otherID.String()

	cases := []struct {
		name, method, path, auth string
		header                   []string
		want                     int
	}{
		{"list without token", http.MethodGet, "/api/staff", "", nil, http.StatusUnauthorized},
		{"list as teacher", http.MethodGet, "/api/staff", teacher, nil, http.StatusForbidden},
		{"instructors as student", http.MethodGet, "/api/staff/instructors?department=x", student, nil, http.StatusForbidden},
		{"create as teacher", http.MethodPost, "/api/staff", teacher, nil, http.StatusForbidden},
		{"update as teacher", http.MethodPut, other, teacher, nil, http.StatusForbidden},
		{"delete as teacher", http.MethodDelete, other, teacher, nil, http.StatusForbidden},
		{"profile edit as teacher", http.MethodPut, other + "/profile", teacher, nil, http.StatusForbidden},
		{"another teacher's record", http.MethodGet, other, teacher, nil, http.StatusForbidden},
		{"internal without secret", http.MethodGet, "/internal/staff/" + s.teacherID.String(), "", nil, http.StatusUnauthorized},
		{"internal with wrong secret", http.MethodGet, "/internal/staff/" + s.teacherID.String(), "", []string{"X-Internal-Secret", "wrong"}, http.StatusUnauthorized},
		{"admin staff create as teacher", http.MethodPost, "/api/admin-staff", teacher, nil, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := s.do(tc.method, tc.path, tc.auth, `{}`, tc.header...)
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
	assert.Empty(t, s.store.events, "a refused request must not reach the outbox")
}

func TestStaffRoutes_OwnRecordAndInternalRead_Allowed(t *testing.T) {
	s := newStaffRoutes(t)

	w := s.do(http.MethodGet, "/api/staff/"+s.teacherID.String(), bearer(t, s.teacherID.String(), "teacher"), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var own dto.StaffResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &own))
	assert.Equal(t, "ahmet@uni.edu.tr", own.Email)

	w = s.do(http.MethodGet, "/internal/staff/"+s.otherID.String(), "", "", "X-Internal-Secret", routesInternalSecret)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestStaffRoutes_AdminLifecycle_WritesThroughOutbox(t *testing.T) {
	s := newStaffRoutes(t)
	admin := bearer(t, uuid.NewString(), "admin")

	w := s.do(http.MethodPost, "/api/staff", admin,
		`{"email":"yeni.hoca@uni.edu.tr","first_name":"Mehmet","last_name":"Demir","role":"teacher","department":"Fizik"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created dto.StaffResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "yeni.hoca@uni.edu.tr", created.Email)
	assert.Equal(t, "Fizik", created.Department)
	require.Len(t, s.store.events, 1, "create must hand its event to the outbox write")
	assert.Equal(t, created.ID, s.store.events[0]["id"])
	assert.Equal(t, "yeni.hoca@uni.edu.tr", s.store.events[0]["email"])

	w = s.do(http.MethodPost, "/api/staff", admin,
		`{"email":"yeni.hoca@uni.edu.tr","first_name":"X","last_name":"Y","role":"teacher","department":"Fizik"}`)
	assert.Equal(t, http.StatusConflict, w.Code, "a second record with the same e-mail")

	w = s.do(http.MethodPut, "/api/staff/"+created.ID, admin, `{"department":"Matematik","office_location":"A-202"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var updated dto.StaffResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, "Matematik", updated.Department)
	assert.Equal(t, "A-202", updated.OfficeLocation)

	w = s.do(http.MethodDelete, "/api/staff/"+created.ID, admin, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.False(t, s.store.rows[uuid.MustParse(created.ID)].IsActive, "delete is a soft delete")
	require.Len(t, s.store.events, 3)
	assert.Equal(t, created.ID, s.store.events[1]["staff_id"], "staff.updated")
	assert.Equal(t, created.ID, s.store.events[2]["staff_id"], "staff.deactivated: auth disables the login by it")

	w = s.do(http.MethodGet, "/api/staff/"+created.ID, admin, "")
	assert.Equal(t, http.StatusNotFound, w.Code, "a deleted record is gone for readers")
	w = s.do(http.MethodDelete, "/api/staff/"+created.ID, admin, "")
	assert.Equal(t, http.StatusNotFound, w.Code, "deleting twice")
}

func TestStaffRoutes_BadInput_Returns400(t *testing.T) {
	s := newStaffRoutes(t)
	admin := bearer(t, uuid.NewString(), "admin")

	cases := []struct{ name, method, path, body string }{
		{"create without e-mail", http.MethodPost, "/api/staff", `{"first_name":"A","last_name":"B","role":"teacher"}`},
		{"create with malformed json", http.MethodPost, "/api/staff", `{"email":`},
		{"update with a bad id", http.MethodPut, "/api/staff/not-a-uuid", `{"department":"X"}`},
		{"profile with a bad id", http.MethodGet, "/api/staff/profile/not-a-uuid", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := s.do(tc.method, tc.path, admin, tc.body)
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}
	assert.Empty(t, s.store.events)
}

func TestStaffRoutes_InstructorsByDepartment_FiltersByDepartment(t *testing.T) {
	s := newStaffRoutes(t)
	admin := bearer(t, uuid.NewString(), "admin")
	w := s.do(http.MethodPost, "/api/staff", admin,
		`{"email":"fizik@uni.edu.tr","first_name":"F","last_name":"Z","role":"teacher","department":"Fizik"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	w = s.do(http.MethodGet, "/internal/staff?department=Bilgisayar%20M%C3%BChendisli%C4%9Fi", "", "", "X-Internal-Secret", routesInternalSecret)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var ids []string
	var generic any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &generic))
	collectIDs(generic, &ids)
	sort.Strings(ids)
	want := []string{s.teacherID.String(), s.otherID.String()}
	sort.Strings(want)
	assert.Equal(t, want, ids, "only the department's two teachers")
}

// collectIDs gathers every "id" value in a decoded JSON document, whatever
// envelope the list comes in.
func collectIDs(v any, ids *[]string) {
	switch x := v.(type) {
	case map[string]any:
		if id, ok := x["id"].(string); ok {
			*ids = append(*ids, id)
		}
		for _, child := range x {
			collectIDs(child, ids)
		}
	case []any:
		for _, child := range x {
			collectIDs(child, ids)
		}
	}
}

func TestAdminStaffRoutes_AdminCreatesAndLists(t *testing.T) {
	s := newStaffRoutes(t)
	admin := bearer(t, uuid.NewString(), "admin")

	w := s.do(http.MethodPost, "/api/admin-staff", admin,
		`{"email":"sekreter@uni.edu.tr","first_name":"Ayşe","last_name":"Kaya","faculty":"Mühendislik","position":"Sekreter","responsibilities":["Evrak"],"start_date":"2020-01-01"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Len(t, s.adminStore.rows, 1)

	w = s.do(http.MethodGet, "/api/admin-staff", admin, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "sekreter@uni.edu.tr")

	w = s.do(http.MethodGet, "/api/admin-staff/"+uuid.NewString(), admin, "")
	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}
