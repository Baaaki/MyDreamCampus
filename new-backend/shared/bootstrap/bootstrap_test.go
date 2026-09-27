package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootstrap_SharedRateLimitBucket(t *testing.T) {
	assert.Equal(t, "public", sharedRateLimitBucket,
		"sharedRateLimitBucket must remain 'public' to prevent per-service rate limit multiplication attacks")
}
