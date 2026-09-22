package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/staff/internal/db"
	"github.com/baaaki/mydreamcampus/staff/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/staff/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if err := logger.Init("test"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

type fakeAdminStaffStore struct {
	rows        map[uuid.UUID]db.AdminStaff
	lastFaculty string
	lastCreate  db.CreateAdminStaffParams
	createErr   error
	updateErr   error
}

func newFakeAdminStaffStore() *fakeAdminStaffStore {
	return &fakeAdminStaffStore{rows: map[uuid.UUID]db.AdminStaff{}}
}

func (f *fakeAdminStaffStore) List(_ context.Context, faculty string) ([]db.AdminStaff, error) {
	f.lastFaculty = faculty
	out := []db.AdminStaff{}
	for _, row := range f.rows {
		if faculty == "" || row.Faculty == faculty {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeAdminStaffStore) GetByID(_ context.Context, id uuid.UUID) (db.AdminStaff, error) {
	row, ok := f.rows[id]
	if !ok {
		return db.AdminStaff{}, fmt.Errorf("%w: admin staff %s", sharedErrors.ErrNotFoundRepo, id)
	}
	return row, nil
}

func (f *fakeAdminStaffStore) Create(_ context.Context, p db.CreateAdminStaffParams) (db.AdminStaff, error) {
	f.lastCreate = p
	if f.createErr != nil {
		return db.AdminStaff{}, f.createErr
	}
	id := uuid.New()
	row := db.AdminStaff{
		ID: utils.UUIDToPgtype(id), Email: p.Email, Title: p.Title,
		FirstName: p.FirstName, LastName: p.LastName, Faculty: p.Faculty,
		Department: p.Department, Phone: p.Phone, ProfileImageUrl: p.ProfileImageUrl,
		Position: p.Position, JobDescription: p.JobDescription,
		Responsibilities: p.Responsibilities, WorkingHours: p.WorkingHours,
		OfficeLocation: p.OfficeLocation, StartDate: p.StartDate, IsActive: true,
	}
	f.rows[id] = row
	return row, nil
}

func (f *fakeAdminStaffStore) Update(_ context.Context, p db.UpdateAdminStaffParams) (db.AdminStaff, error) {
	if f.updateErr != nil {
		return db.AdminStaff{}, f.updateErr
	}
	id := utils.PgtypeToUUID(p.ID)
	row, ok := f.rows[id]
	if !ok {
		return db.AdminStaff{}, fmt.Errorf("%w: admin staff for update", sharedErrors.ErrNotFoundRepo)
	}
	row.Email, row.Position, row.Responsibilities = p.Email, p.Position, p.Responsibilities
	f.rows[id] = row
	return row, nil
}

func validAdminStaffRequest() dto.AdminStaffRequest {
	return dto.AdminStaffRequest{
		Email:            "ali.vural@uni.edu.tr",
		FirstName:        "Ali",
		LastName:         "Vural",
		Faculty:          "Mühendislik Fakültesi",
		Position:         "Fakülte Sekreteri",
		Responsibilities: []string{"Yazışmalar", "Kurul toplantıları"},
		StartDate:        "2019-09-01",
	}
}

func TestAdminStaffService_Create_ValidRequest_StoresAndReturnsRecord(t *testing.T) {
	store := newFakeAdminStaffStore()
	svc := NewAdminStaffService(store)

	resp, err := svc.Create(context.Background(), validAdminStaffRequest())
	require.NoError(t, err)

	var stored []string
	require.NoError(t, json.Unmarshal(store.lastCreate.Responsibilities, &stored))
	assert.Equal(t, []string{"Yazışmalar", "Kurul toplantıları"}, stored)
	assert.Equal(t, pgtype.Date{Time: time.Date(2019, 9, 1, 0, 0, 0, 0, time.UTC), Valid: true}, store.lastCreate.StartDate)

	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "2019-09-01", resp.StartDate)
	assert.Equal(t, []string{"Yazışmalar", "Kurul toplantıları"}, resp.Responsibilities)
	assert.Equal(t, "active", resp.Status)
}

func TestAdminStaffService_Create_NoResponsibilities_StoresEmptyArray(t *testing.T) {
	store := newFakeAdminStaffStore()
	req := validAdminStaffRequest()
	req.Responsibilities = nil
	req.StartDate = ""

	resp, err := NewAdminStaffService(store).Create(context.Background(), req)
	require.NoError(t, err)

	// A JSON null would violate the column's NOT NULL '[]' contract.
	assert.JSONEq(t, `[]`, string(store.lastCreate.Responsibilities))
	assert.False(t, store.lastCreate.StartDate.Valid)
	assert.Equal(t, []string{}, resp.Responsibilities)
	assert.Empty(t, resp.StartDate)
}

func TestAdminStaffService_Create_DuplicateEmail_ReturnsEmailExists(t *testing.T) {
	store := newFakeAdminStaffStore()
	store.createErr = fmt.Errorf("%w: admin staff email already exists", sharedErrors.ErrAlreadyExistsRepo)

	_, err := NewAdminStaffService(store).Create(context.Background(), validAdminStaffRequest())

	assert.ErrorIs(t, err, serviceErrors.ErrEmailExists)
}

func TestAdminStaffService_Get_UnknownID_ReturnsNotFound(t *testing.T) {
	_, err := NewAdminStaffService(newFakeAdminStaffStore()).Get(context.Background(), uuid.NewString())

	assert.ErrorIs(t, err, serviceErrors.ErrAdminStaffNotFound)
}

func TestAdminStaffService_Get_MalformedID_ReturnsInvalidIDWithoutQuerying(t *testing.T) {
	_, err := NewAdminStaffService(newFakeAdminStaffStore()).Get(context.Background(), "not-a-uuid")

	assert.ErrorIs(t, err, sharedErrors.ErrInvalidID)
}

func TestAdminStaffService_Update_UnknownID_ReturnsNotFound(t *testing.T) {
	_, err := NewAdminStaffService(newFakeAdminStaffStore()).
		Update(context.Background(), uuid.NewString(), validAdminStaffRequest())

	assert.ErrorIs(t, err, serviceErrors.ErrAdminStaffNotFound)
}

func TestAdminStaffService_Update_StoreFailure_ReturnsInternal(t *testing.T) {
	store := newFakeAdminStaffStore()
	store.updateErr = fmt.Errorf("%w: connection reset", sharedErrors.ErrQueryFailed)

	_, err := NewAdminStaffService(store).Update(context.Background(), uuid.NewString(), validAdminStaffRequest())

	assert.ErrorIs(t, err, sharedErrors.ErrInternal)
}

func TestAdminStaffService_List_FacultyFilter_ReturnsOnlyThatFaculty(t *testing.T) {
	store := newFakeAdminStaffStore()
	svc := NewAdminStaffService(store)
	_, err := svc.Create(context.Background(), validAdminStaffRequest())
	require.NoError(t, err)
	other := validAdminStaffRequest()
	other.Email, other.Faculty = "hasan.yildiz@uni.edu.tr", "Fen Fakültesi"
	_, err = svc.Create(context.Background(), other)
	require.NoError(t, err)

	resp, err := svc.List(context.Background(), "Fen Fakültesi")
	require.NoError(t, err)

	assert.Equal(t, "Fen Fakültesi", store.lastFaculty)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "hasan.yildiz@uni.edu.tr", resp.Data[0].Email)
}

func TestAdminStaffService_Read_CorruptResponsibilities_DegradesToEmptyList(t *testing.T) {
	store := newFakeAdminStaffStore()
	id := uuid.New()
	store.rows[id] = db.AdminStaff{ID: utils.UUIDToPgtype(id), IsActive: true, Responsibilities: []byte(`{"not":"a list"}`)}

	resp, err := NewAdminStaffService(store).Get(context.Background(), id.String())

	require.NoError(t, err)
	assert.Equal(t, []string{}, resp.Responsibilities)
}
