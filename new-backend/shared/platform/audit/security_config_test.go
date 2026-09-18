package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityConfig_NeverSamples(t *testing.T) {
	for _, env := range []string{"production", "development"} {
		assert.Nil(t, securityConfig(env).Sampling,
			"audit entries all share one message, so sampling would drop most of a burst (%s)", env)
	}
}
