package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	serviceErrors "github.com/baaaki/mydreamcampus/meal/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The web and mobile clients read {error: message, code: CODE} from every
// service; meal used to wrap it as {success, error: {code, message}}.
func TestHandleError_CommonErrorShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &MealHandler{logger: zap.NewNop()}

	cases := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{serviceErrors.ErrReservationAlreadyUsed, http.StatusBadRequest, "RESERVATION_ALREADY_USED"},
		{serviceErrors.ErrReservationConflicts, http.StatusConflict, "RESERVATION_CONFLICTS"},
		{errors.New("boom"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		h.handleError(c, tc.err)

		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, tc.wantStatus, w.Code, tc.wantCode)
		assert.Equal(t, tc.wantCode, body["code"])
		assert.IsType(t, "", body["error"], "error must be the message string")
		assert.NotEmpty(t, body["error"])
		assert.NotContains(t, body, "success")
	}
}
