package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	"github.com/baaaki/mydreamcampus/catalog/internal/service"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFacultyReader struct {
	faculties   []db.Faculty
	departments []db.Department
	err         error
}

func (s stubFacultyReader) ListFaculties(context.Context) ([]db.Faculty, error) {
	return s.faculties, s.err
}

func (s stubFacultyReader) ListDepartments(context.Context) ([]db.Department, error) {
	return s.departments, s.err
}

func serveFaculties(t *testing.T, reader service.FacultyReader) *httptest.ResponseRecorder {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/catalog/faculties", NewFacultyHandler(service.NewFacultyService(reader)).ListFaculties)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/catalog/faculties", nil))
	return w
}

// The frontend's Faculty/Department types are the contract: slug ids and a
// camelCase facultyId. A snake_case key here would leave every filter empty.
func TestListFaculties_Success_ReturnsFrontendFacultyShape(t *testing.T) {
	fen := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	w := serveFaculties(t, stubFacultyReader{
		faculties:   []db.Faculty{{ID: fen, Slug: "fac-fen", Code: "FEN", Name: "Fen Fakültesi"}},
		departments: []db.Department{{FacultyID: fen, Slug: "dept-fizik", Code: "FIZ", Name: "Fizik"}},
	})

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string][]map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body["data"], 1)

	faculty := body["data"][0]
	assert.Equal(t, "fac-fen", faculty["id"])
	assert.Equal(t, "Fen Fakültesi", faculty["name"])
	assert.Equal(t, "FEN", faculty["code"])

	departments, ok := faculty["departments"].([]any)
	require.True(t, ok)
	require.Len(t, departments, 1)
	dept := departments[0].(map[string]any)
	assert.Equal(t, map[string]any{
		"id":        "dept-fizik",
		"name":      "Fizik",
		"facultyId": "fac-fen",
		"code":      "FIZ",
	}, dept, "description is omitted when empty")
}

func TestListFaculties_QueryFailure_Returns500(t *testing.T) {
	w := serveFaculties(t, stubFacultyReader{err: errors.New("connection refused")})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "connection refused", "the cause stays in the log")
}
