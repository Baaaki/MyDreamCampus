package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	"github.com/baaaki/mydreamcampus/meal/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/meal/internal/errors"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeCafeterias answers like repository.CafeteriaRepository: an unknown id
// is ErrCafeteriaNotFoundRepo.
type fakeCafeterias struct {
	rows map[uuid.UUID]db.Cafeteria
}

func (f *fakeCafeterias) notFound(id uuid.UUID) error {
	return fmt.Errorf("%w: cafeteria %s", serviceErrors.ErrCafeteriaNotFoundRepo, id)
}

func (f *fakeCafeterias) list(onlyActive bool) []db.Cafeteria {
	var out []db.Cafeteria
	for _, c := range f.rows {
		if c.IsActive || !onlyActive {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeCafeterias) GetActiveCafeterias(context.Context) ([]db.Cafeteria, error) {
	return f.list(true), nil
}

func (f *fakeCafeterias) GetAllCafeterias(context.Context) ([]db.Cafeteria, error) {
	return f.list(false), nil
}

func (f *fakeCafeterias) GetCafeteriaByID(_ context.Context, id uuid.UUID) (db.Cafeteria, error) {
	c, ok := f.rows[id]
	if !ok {
		return c, f.notFound(id)
	}
	return c, nil
}

func (f *fakeCafeterias) CreateCafeteria(_ context.Context, p db.CreateCafeteriaParams) (db.Cafeteria, error) {
	id := uuid.New()
	c := db.Cafeteria{ID: utils.UUIDToPgtype(id), Name: p.Name, Location: p.Location, HasVeganMenu: p.HasVeganMenu, ServesDinner: p.ServesDinner, IsActive: p.IsActive}
	f.rows[id] = c
	return c, nil
}

func (f *fakeCafeterias) UpdateCafeteria(_ context.Context, p db.UpdateCafeteriaParams) (db.Cafeteria, error) {
	id := utils.PgtypeToUUID(p.ID)
	if _, ok := f.rows[id]; !ok {
		return db.Cafeteria{}, f.notFound(id)
	}
	c := db.Cafeteria{ID: p.ID, Name: p.Name, Location: p.Location, HasVeganMenu: p.HasVeganMenu, ServesDinner: p.ServesDinner, IsActive: p.IsActive}
	f.rows[id] = c
	return c, nil
}

func (f *fakeCafeterias) DeactivateCafeteria(_ context.Context, id uuid.UUID) error {
	c, ok := f.rows[id]
	if !ok {
		return f.notFound(id)
	}
	c.IsActive = false
	f.rows[id] = c
	return nil
}

func TestCafeteriaService_Lifecycle(t *testing.T) {
	store := &fakeCafeterias{rows: map[uuid.UUID]db.Cafeteria{}}
	svc := NewCafeteriaService(store, zap.NewNop())
	ctx := context.Background()

	created, err := svc.CreateCafeteria(ctx, dto.CreateCafeteriaRequest{Name: "Merkez", Location: "A Blok", ServesDinner: true, IsActive: true})
	require.NoError(t, err)
	_, err = svc.CreateCafeteria(ctx, dto.CreateCafeteriaRequest{Name: "Kapalı", IsActive: false})
	require.NoError(t, err)

	active, err := svc.GetActiveCafeterias(ctx)
	require.NoError(t, err)
	assert.Len(t, active.Cafeterias, 1, "students only see open cafeterias")
	all, err := svc.GetAllCafeterias(ctx)
	require.NoError(t, err)
	assert.Len(t, all.Cafeterias, 2, "admins see every cafeteria")

	updated, err := svc.UpdateCafeteria(ctx, created.ID, dto.UpdateCafeteriaRequest{Name: "Merkez Yemekhane", HasVeganMenu: true, IsActive: true})
	require.NoError(t, err)
	assert.True(t, updated.HasVeganMenu)

	require.NoError(t, svc.DeactivateCafeteria(ctx, created.ID))
	got, err := svc.GetCafeteriaByID(ctx, created.ID)
	require.NoError(t, err)
	assert.False(t, got.IsActive, "deactivation keeps the row for past reservations")
}

func TestCafeteriaService_BadOrUnknownID(t *testing.T) {
	svc := NewCafeteriaService(&fakeCafeterias{rows: map[uuid.UUID]db.Cafeteria{}}, zap.NewNop())
	ctx := context.Background()
	unknown := uuid.NewString()

	_, err := svc.GetCafeteriaByID(ctx, "a-blok")
	assert.ErrorIs(t, err, sharedErrors.ErrBadRequest)
	_, err = svc.GetCafeteriaByID(ctx, unknown)
	assert.ErrorIs(t, err, serviceErrors.ErrCafeteriaNotFound)
	_, err = svc.UpdateCafeteria(ctx, unknown, dto.UpdateCafeteriaRequest{Name: "X"})
	assert.ErrorIs(t, err, serviceErrors.ErrCafeteriaNotFound)
	assert.ErrorIs(t, svc.DeactivateCafeteria(ctx, unknown), serviceErrors.ErrCafeteriaNotFound)
	assert.ErrorIs(t, svc.DeactivateCafeteria(ctx, "a-blok"), sharedErrors.ErrBadRequest)
}

// fakeMenus answers like repository.MenuRepository: a month with no menu is
// ErrNotFoundRepo.
type fakeMenus struct {
	rows map[string]db.MonthlyMenu
}

func (f *fakeMenus) UpsertMonthlyMenu(_ context.Context, p db.UpsertMonthlyMenuParams) (db.MonthlyMenu, error) {
	m := db.MonthlyMenu{Year: p.Year, Month: p.Month, MenuData: p.MenuData}
	f.rows[fmt.Sprint(p.Year, p.Month)] = m
	return m, nil
}

func (f *fakeMenus) GetMonthlyMenu(_ context.Context, p db.GetMonthlyMenuParams) (db.MonthlyMenu, error) {
	m, ok := f.rows[fmt.Sprint(p.Year, p.Month)]
	if !ok {
		return m, fmt.Errorf("%w: menu %d-%d", sharedErrors.ErrNotFoundRepo, p.Year, p.Month)
	}
	return m, nil
}

func TestMenuService_SaveThenReadCurrentMonth(t *testing.T) {
	freezeAt(t, 2026, time.October, 5, 9, 0)
	svc := NewMenuService(&fakeMenus{rows: map[string]db.MonthlyMenu{}}, zap.NewNop())
	ctx := context.Background()
	day := map[string]any{"2026-10-05": map[string]any{"lunch": map[string]any{"normal": []any{"Mercimek çorbası"}}}}

	_, err := svc.GetMonthlyMenu(ctx, 0, 0)
	assert.ErrorIs(t, err, sharedErrors.ErrNotFound, "no menu published yet")

	saved, err := svc.CreateOrUpdateMonthlyMenu(ctx, dto.CreateMonthlyMenuRequest{Year: 2026, Month: 10, MenuData: day})
	require.NoError(t, err)
	assert.Equal(t, 10, saved.Month)

	current, err := svc.GetMonthlyMenu(ctx, 0, 0)
	require.NoError(t, err, "no year or month means the current one")
	want, _ := json.Marshal(day)
	got, _ := json.Marshal(current.MenuData)
	assert.JSONEq(t, string(want), string(got))
}
