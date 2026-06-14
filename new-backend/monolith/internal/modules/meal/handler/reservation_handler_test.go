package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// CreateReservationRequest enforces enum bounds at the binding layer. The
// service layer also validates against per-cafeteria capabilities (see
// service.validateMealTimeAndMenu) — these tests cover only the HTTP-layer
// rules; deeper checks are in service/reservation_validators_test.go.

func TestCreateReservationRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cafID := uuid.New().String()

	tests := []struct {
		name           string
		body           any
		expectedStatus int
	}{
		{
			name: "valid lunch normal",
			body: dto.CreateReservationRequest{
				CafeteriaID: cafID,
				Date:        "2026-04-28",
				MealTime:    "lunch",
				MenuType:    "normal",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "non-uuid cafeteria_id is rejected",
			body: dto.CreateReservationRequest{
				CafeteriaID: "not-a-uuid",
				Date:        "2026-04-28",
				MealTime:    "lunch",
				MenuType:    "normal",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "meal_time outside enum",
			body: dto.CreateReservationRequest{
				CafeteriaID: cafID,
				Date:        "2026-04-28",
				MealTime:    "brunch",
				MenuType:    "normal",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "menu_type outside enum",
			body: dto.CreateReservationRequest{
				CafeteriaID: cafID,
				Date:        "2026-04-28",
				MealTime:    "lunch",
				MenuType:    "keto",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "non-JSON body",
			body:           "{not valid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/reservations", func(c *gin.Context) {
				var req dto.CreateReservationRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			var raw []byte
			if s, ok := tc.body.(string); ok {
				raw = []byte(s)
			} else {
				raw, _ = json.Marshal(tc.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/reservations", bytes.NewBuffer(raw))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedStatus, resp.Code)
		})
	}
}
