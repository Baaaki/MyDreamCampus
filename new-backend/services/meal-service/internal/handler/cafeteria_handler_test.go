package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/meal/internal/db"
	"github.com/baaaki/mydreamcampus/meal/internal/service"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubCafeteriaStore struct {
	service.CafeteriaStore
	cafeterias []db.Cafeteria
}

func (s *stubCafeteriaStore) GetActiveCafeterias(ctx context.Context) ([]db.Cafeteria, error) {
	return s.cafeterias, nil
}

func TestGetCafeterias_Student_IncludesMealPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Reservation: config.ReservationConfig{MealPriceTRY: 15}}
	store := &stubCafeteriaStore{cafeterias: []db.Cafeteria{{
		ID:       pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Name:     "Merkez Yemekhane",
		Location: "Kampüs",
		IsActive: true,
	}}}
	h := NewMealHandler(
		service.NewCafeteriaService(store, zap.NewNop()),
		service.NewReservationService(nil, nil, nil, nil, nil, cfg, zap.NewNop()),
		nil,
		zap.NewNop(),
	)
	router := gin.New()
	router.GET("/cafeterias", func(c *gin.Context) {
		c.Set("role", "student")
		c.Next()
	}, h.GetCafeterias)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/cafeterias", nil))

	require.Equal(t, http.StatusOK, resp.Code)
	var body struct {
		Data struct {
			Cafeterias []map[string]any `json:"cafeterias"`
			MealPrice  float64          `json:"meal_price"`
			Currency   string           `json:"currency"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Len(t, body.Data.Cafeterias, 1)
	assert.Equal(t, 15.0, body.Data.MealPrice)
	assert.Equal(t, "TRY", body.Data.Currency)
}
