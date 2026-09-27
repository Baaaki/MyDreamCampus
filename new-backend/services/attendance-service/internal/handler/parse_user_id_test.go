package handler

import (
	"net/http/httptest"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing user_id returns unauthorized", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		uid, err := parseUserID(c)
		assert.Equal(t, uuid.Nil, uid)
		assert.ErrorIs(t, err, sharedErrors.ErrUnauthorized)
	})

	t.Run("valid uuid string returns parsed UUID", func(t *testing.T) {
		expected := uuid.New()
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", expected.String())

		uid, err := parseUserID(c)
		require.NoError(t, err)
		assert.Equal(t, expected, uid)
	})

	t.Run("valid uuid.UUID object returns directly", func(t *testing.T) {
		expected := uuid.New()
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", expected)

		uid, err := parseUserID(c)
		require.NoError(t, err)
		assert.Equal(t, expected, uid)
	})

	t.Run("invalid type returns unauthorized", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", 12345)

		uid, err := parseUserID(c)
		assert.Equal(t, uuid.Nil, uid)
		assert.ErrorIs(t, err, sharedErrors.ErrUnauthorized)
	})
}
