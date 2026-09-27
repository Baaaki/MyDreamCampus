package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockClosedDaysStore struct {
	days map[uuid.UUID]db.ClosedDay
	err  error
}

func newMockClosedDaysStore() *mockClosedDaysStore {
	return &mockClosedDaysStore{
		days: make(map[uuid.UUID]db.ClosedDay),
	}
}

func (m *mockClosedDaysStore) CreateClosedDay(ctx context.Context, params db.CreateClosedDayParams) (db.ClosedDay, error) {
	if m.err != nil {
		return db.ClosedDay{}, m.err
	}
	id := uuid.New()
	cd := db.ClosedDay{
		ID:        utils.UUIDToPgtype(id),
		Date:      params.Date,
		Reason:    params.Reason,
		Semester:  params.Semester,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	m.days[id] = cd
	return cd, nil
}

func (m *mockClosedDaysStore) DeleteClosedDay(ctx context.Context, id pgtype.UUID) error {
	if m.err != nil {
		return m.err
	}
	u := utils.PgtypeToUUID(id)
	delete(m.days, u)
	return nil
}

func (m *mockClosedDaysStore) ListClosedDays(ctx context.Context, params db.ListClosedDaysParams) ([]db.ClosedDay, error) {
	if m.err != nil {
		return nil, m.err
	}
	var res []db.ClosedDay
	for _, v := range m.days {
		res = append(res, v)
	}
	return res, nil
}

func (m *mockClosedDaysStore) GetClosedDaysByDateRange(ctx context.Context, params db.GetClosedDaysByDateRangeParams) ([]db.ClosedDay, error) {
	return m.ListClosedDays(ctx, db.ListClosedDaysParams{})
}

func (m *mockClosedDaysStore) DeleteClosedDaysBySemester(ctx context.Context, semester string) error {
	return m.err
}

func TestClosedDaysHandler_CreateClosedDay(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful creation", func(t *testing.T) {
		store := newMockClosedDaysStore()
		h := NewClosedDaysHandler(store, zap.NewNop(), nil)
		r := gin.New()
		r.POST("/closed-days", h.CreateClosedDay)

		body := `{"date":"2026-10-29","reason":"Cumhuriyet Bayramı"}`
		req := httptest.NewRequest(http.MethodPost, "/closed-days", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, 1, len(store.days))
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		store := newMockClosedDaysStore()
		h := NewClosedDaysHandler(store, zap.NewNop(), nil)
		r := gin.New()
		r.POST("/closed-days", h.CreateClosedDay)

		body := `{"reason":"Cumhuriyet Bayramı"}` // missing date
		req := httptest.NewRequest(http.MethodPost, "/closed-days", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid date format returns 400", func(t *testing.T) {
		store := newMockClosedDaysStore()
		h := NewClosedDaysHandler(store, zap.NewNop(), nil)
		r := gin.New()
		r.POST("/closed-days", h.CreateClosedDay)

		body := `{"date":"29-10-2026","reason":"Cumhuriyet Bayramı"}` // DD-MM-YYYY instead of YYYY-MM-DD
		req := httptest.NewRequest(http.MethodPost, "/closed-days", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("conflict error returns 409", func(t *testing.T) {
		store := newMockClosedDaysStore()
		store.err = errors.New("unique violation")
		h := NewClosedDaysHandler(store, zap.NewNop(), nil)
		r := gin.New()
		r.POST("/closed-days", h.CreateClosedDay)

		body := `{"date":"2026-10-29","reason":"Cumhuriyet Bayramı"}`
		req := httptest.NewRequest(http.MethodPost, "/closed-days", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestClosedDaysHandler_ListClosedDays(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid list without filters", func(t *testing.T) {
		store := newMockClosedDaysStore()
		h := NewClosedDaysHandler(store, zap.NewNop(), nil)
		r := gin.New()
		r.GET("/closed-days", h.ListClosedDays)

		req := httptest.NewRequest(http.MethodGet, "/closed-days", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid from date filter returns 400", func(t *testing.T) {
		store := newMockClosedDaysStore()
		h := NewClosedDaysHandler(store, zap.NewNop(), nil)
		r := gin.New()
		r.GET("/closed-days", h.ListClosedDays)

		req := httptest.NewRequest(http.MethodGet, "/closed-days?from=invalid-date", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestClosedDaysHandler_DeleteClosedDay(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := newMockClosedDaysStore()
		h := NewClosedDaysHandler(store, zap.NewNop(), nil)
		r := gin.New()
		r.DELETE("/closed-days/:id", h.DeleteClosedDay)

		req := httptest.NewRequest(http.MethodDelete, "/closed-days/not-a-uuid", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid uuid returns 200", func(t *testing.T) {
		store := newMockClosedDaysStore()
		h := NewClosedDaysHandler(store, zap.NewNop(), nil)
		r := gin.New()
		r.DELETE("/closed-days/:id", h.DeleteClosedDay)

		validID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/closed-days/"+validID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
