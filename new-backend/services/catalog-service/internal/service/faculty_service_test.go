package service

import (
	"context"
	"errors"
	"testing"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFacultyReader struct {
	faculties   []db.Faculty
	departments []db.Department
	err         error
}

func (f fakeFacultyReader) ListFaculties(context.Context) ([]db.Faculty, error) {
	return f.faculties, f.err
}

func (f fakeFacultyReader) ListDepartments(context.Context) ([]db.Department, error) {
	return f.departments, f.err
}

func pgID() pgtype.UUID {
	return pgtype.UUID{Bytes: uuid.New(), Valid: true}
}

func TestFacultyService_ListFaculties_NestsDepartmentsUnderFacultySlug(t *testing.T) {
	egitim, fen, hukuk := pgID(), pgID(), pgID()
	reader := fakeFacultyReader{
		faculties: []db.Faculty{
			{ID: egitim, Slug: "fac-egitim", Code: "BEF", Name: "Eğitim Fakültesi"},
			{ID: fen, Slug: "fac-fen", Code: "FEN", Name: "Fen Fakültesi"},
			{ID: hukuk, Slug: "fac-hukuk", Code: "HKK", Name: "Hukuk Fakültesi"},
		},
		departments: []db.Department{
			{FacultyID: fen, Slug: "dept-bil", Code: "BIL", Name: "Bilgisayar Bilimleri",
				Description: pgtype.Text{String: "Bilgisayar Bilimleri Lisans Programı", Valid: true}},
			{FacultyID: egitim, Slug: "dept-bote", Code: "BOTE", Name: "Bilgisayar ve Öğretim Teknolojileri Öğretmenliği"},
			{FacultyID: fen, Slug: "dept-fizik", Code: "FIZ", Name: "Fizik"},
		},
	}

	resp, err := NewFacultyService(reader).ListFaculties(context.Background())
	require.NoError(t, err)
	require.Len(t, resp.Data, 3)

	assert.Equal(t, "fac-egitim", resp.Data[0].ID)
	require.Len(t, resp.Data[0].Departments, 1)
	assert.Equal(t, "dept-bote", resp.Data[0].Departments[0].ID)
	assert.Equal(t, "fac-egitim", resp.Data[0].Departments[0].FacultyID)
	assert.Nil(t, resp.Data[0].Departments[0].Description)

	fenResp := resp.Data[1]
	assert.Equal(t, "fac-fen", fenResp.ID)
	require.Len(t, fenResp.Departments, 2)
	assert.Equal(t, []string{"dept-bil", "dept-fizik"},
		[]string{fenResp.Departments[0].ID, fenResp.Departments[1].ID}, "department order is kept")
	require.NotNil(t, fenResp.Departments[0].Description)
	assert.Equal(t, "Bilgisayar Bilimleri Lisans Programı", *fenResp.Departments[0].Description)

	assert.NotNil(t, resp.Data[2].Departments, "a faculty without departments serialises as []")
	assert.Empty(t, resp.Data[2].Departments)
}

func TestFacultyService_ListFaculties_OrphanDepartmentIsDropped(t *testing.T) {
	fen := pgID()
	reader := fakeFacultyReader{
		faculties:   []db.Faculty{{ID: fen, Slug: "fac-fen", Name: "Fen Fakültesi"}},
		departments: []db.Department{{FacultyID: pgID(), Slug: "dept-orphan", Name: "Yetim"}},
	}

	resp, err := NewFacultyService(reader).ListFaculties(context.Background())
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Empty(t, resp.Data[0].Departments)
}

func TestFacultyService_ListFaculties_QueryFailure_ReturnsInternal(t *testing.T) {
	reader := fakeFacultyReader{err: errors.New("connection refused")}

	_, err := NewFacultyService(reader).ListFaculties(context.Background())
	require.Error(t, err)
	assert.True(t, sharedErrors.Is(err, sharedErrors.ErrInternal))
}
