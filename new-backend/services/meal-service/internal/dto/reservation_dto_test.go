package dto

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validateBinding[T any](t *testing.T, body any) (int, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/", func(c *gin.Context) {
		var got T
		if err := c.ShouldBindJSON(&got); err != nil {
			c.AbortWithStatusJSON(400, gin.H{"err": err.Error()})
			return
		}
		c.Status(200)
	})
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func TestCreateReservationRequest_Validation(t *testing.T) {
	valid := map[string]any{
		"cafeteria_id": uuid.NewString(),
		"date":         "2026-04-26",
		"meal_time":    "lunch",
		"menu_type":    "normal",
	}

	t.Run("happy path", func(t *testing.T) {
		code, _ := validateBinding[CreateReservationRequest](t, valid)
		assert.Equal(t, 200, code)
	})

	t.Run("invalid cafeteria_id rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["cafeteria_id"] = "not-a-uuid"
		code, _ := validateBinding[CreateReservationRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("invalid meal_time rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["meal_time"] = "snack"
		code, _ := validateBinding[CreateReservationRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("invalid menu_type rejected", func(t *testing.T) {
		body := copyMap(valid)
		body["menu_type"] = "kosher"
		code, _ := validateBinding[CreateReservationRequest](t, body)
		assert.Equal(t, 400, code)
	})

	t.Run("missing date rejected", func(t *testing.T) {
		body := copyMap(valid)
		delete(body, "date")
		code, _ := validateBinding[CreateReservationRequest](t, body)
		assert.Equal(t, 400, code)
	})
}

func TestBatchReservationRequest_Bounds(t *testing.T) {
	one := map[string]any{
		"cafeteria_id": uuid.NewString(),
		"date":         "2026-04-26",
		"meal_time":    "lunch",
		"menu_type":    "normal",
	}

	t.Run("empty list rejected", func(t *testing.T) {
		code, _ := validateBinding[BatchReservationRequest](t,
			map[string]any{"reservations": []map[string]any{}})
		assert.Equal(t, 400, code)
	})

	t.Run("more than 10 rejected", func(t *testing.T) {
		batch := make([]map[string]any, 11)
		for i := range batch {
			batch[i] = one
		}
		code, _ := validateBinding[BatchReservationRequest](t,
			map[string]any{"reservations": batch})
		assert.Equal(t, 400, code)
	})

	t.Run("up to 10 accepted", func(t *testing.T) {
		batch := make([]map[string]any, 10)
		for i := range batch {
			batch[i] = one
		}
		code, _ := validateBinding[BatchReservationRequest](t,
			map[string]any{"reservations": batch})
		assert.Equal(t, 200, code)
	})

	t.Run("dive validates each entry", func(t *testing.T) {
		bad := copyMap(one)
		bad["meal_time"] = "snack"
		code, _ := validateBinding[BatchReservationRequest](t,
			map[string]any{"reservations": []map[string]any{one, bad}})
		assert.Equal(t, 400, code)
	})
}

func TestUseReservationRequest_RequiresQRPayload(t *testing.T) {
	code, _ := validateBinding[UseReservationRequest](t, map[string]any{})
	assert.Equal(t, 400, code)

	code, _ = validateBinding[UseReservationRequest](t,
		map[string]any{"qr_payload": "abc:def"})
	assert.Equal(t, 200, code)
}

func TestReservationResponse_OmitsCafeteriaWhenNil(t *testing.T) {
	r := ReservationResponse{ID: "r-1", Status: "pending"}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"cafeteria":`)
}

func TestReservationSummary_AllZeroSerializes(t *testing.T) {
	s := ReservationSummary{}
	data, err := json.Marshal(s)
	require.NoError(t, err)
	str := string(data)
	for _, k := range []string{"total", "confirmed", "pending", "used", "cancelled"} {
		assert.Contains(t, str, k)
	}
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
