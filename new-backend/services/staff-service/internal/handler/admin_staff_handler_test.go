package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/staff/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/staff/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAdminStaffService struct {
	err         error
	created     *dto.AdminStaffRequest
	listFaculty string
}

func (f *fakeAdminStaffService) List(_ context.Context, faculty string) (dto.AdminStaffListResponse, error) {
	f.listFaculty = faculty
	return dto.AdminStaffListResponse{Data: []dto.AdminStaffResponse{}}, f.err
}

func (f *fakeAdminStaffService) Get(_ context.Context, id string) (dto.AdminStaffResponse, error) {
	return dto.AdminStaffResponse{ID: id}, f.err
}

func (f *fakeAdminStaffService) Create(_ context.Context, req dto.AdminStaffRequest) (dto.AdminStaffResponse, error) {
	f.created = &req
	return dto.AdminStaffResponse{Email: req.Email}, f.err
}

func (f *fakeAdminStaffService) Update(_ context.Context, id string, req dto.AdminStaffRequest) (dto.AdminStaffResponse, error) {
	return dto.AdminStaffResponse{ID: id, Email: req.Email}, f.err
}

func newAdminStaffRouter(t *testing.T, svc *fakeAdminStaffService) *gin.Engine {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	h := NewAdminStaffHandler(svc)
	r := gin.New()
	r.GET("/admin-staff", h.List)
	r.GET("/admin-staff/:id", h.Get)
	r.POST("/admin-staff", h.Create)
	r.PUT("/admin-staff/:id", h.Update)
	return r
}

func adminStaffBody(mutate func(map[string]any)) *bytes.Reader {
	body := map[string]any{
		"email":            "ali.vural@uni.edu.tr",
		"first_name":       "Ali",
		"last_name":        "Vural",
		"faculty":          "Mühendislik Fakültesi",
		"position":         "Fakülte Sekreteri",
		"responsibilities": []string{"Yazışmalar"},
		"start_date":       "2019-09-01",
	}
	if mutate != nil {
		mutate(body)
	}
	raw, _ := json.Marshal(body)
	return bytes.NewReader(raw)
}

func TestAdminStaffHandler_Create_ValidBody_Returns201(t *testing.T) {
	svc := &fakeAdminStaffService{}
	r := newAdminStaffRouter(t, svc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin-staff", adminStaffBody(nil)))

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, svc.created)
	assert.Equal(t, []string{"Yazışmalar"}, svc.created.Responsibilities)
}

func TestAdminStaffHandler_Create_InvalidBody_Returns400(t *testing.T) {
	cases := map[string]func(map[string]any){
		"bad email":               func(b map[string]any) { b["email"] = "not-an-email" },
		"missing position":        func(b map[string]any) { delete(b, "position") },
		"javascript image url":    func(b map[string]any) { b["profile_image_url"] = "javascript:alert(1)" },
		"data image url":          func(b map[string]any) { b["profile_image_url"] = "data:image/png;base64,AAAA" },
		"start date not ISO":      func(b map[string]any) { b["start_date"] = "01.09.2019" },
		"responsibility too long": func(b map[string]any) { b["responsibilities"] = []string{strings.Repeat("a", 301)} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			svc := &fakeAdminStaffService{}
			r := newAdminStaffRouter(t, svc)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin-staff", adminStaffBody(mutate)))

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Nil(t, svc.created, "an invalid body must not reach the service")
		})
	}
}

func TestAdminStaffHandler_Create_HTTPSImageURL_Accepted(t *testing.T) {
	svc := &fakeAdminStaffService{}
	r := newAdminStaffRouter(t, svc)

	w := httptest.NewRecorder()
	body := adminStaffBody(func(b map[string]any) { b["profile_image_url"] = "https://cdn.uni.edu.tr/a.jpg" })
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin-staff", body))

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAdminStaffHandler_Errors_MapToStatusAndCode(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"not found", serviceErrors.ErrAdminStaffNotFound, http.StatusNotFound, "ADMIN_STAFF_NOT_FOUND"},
		{"duplicate email", serviceErrors.ErrEmailExists, http.StatusConflict, "EMAIL_EXISTS"},
		{"bad id", sharedErrors.ErrInvalidID, http.StatusBadRequest, "INVALID_ID"},
		{"internal keeps detail out", sharedErrors.Wrap(sharedErrors.ErrInternal, errors.New("pq: secret detail")), http.StatusInternalServerError, "INTERNAL_ERROR"},
		{"non-AppError is a 500", errors.New("boom"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newAdminStaffRouter(t, &fakeAdminStaffService{err: tc.err})

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/admin-staff/abc", adminStaffBody(nil)))

			assert.Equal(t, tc.status, w.Code)
			var resp dto.ErrorResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tc.code, resp.Code)
			assert.NotContains(t, w.Body.String(), "secret detail")
		})
	}
}

func TestAdminStaffHandler_List_PassesFacultyFilter(t *testing.T) {
	svc := &fakeAdminStaffService{}
	r := newAdminStaffRouter(t, svc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin-staff?faculty=Fen+Fak%C3%BCltesi", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Fen Fakültesi", svc.listFaculty)
	assert.JSONEq(t, `{"data":[]}`, w.Body.String())
}
