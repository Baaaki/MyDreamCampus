package eventbus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModuleExchanges_CompleteAndUnique(t *testing.T) {
	expected := []string{
		"auth.events",
		"staff.events",
		"student.events",
		"course_catalog.events",
		"enrollment.events",
		"attendance.events",
		"grades.events",
		"meal.events",
		"payment.events",
	}

	assert.Equal(t, len(expected), len(ModuleExchanges))

	seen := make(map[string]bool)
	for _, ex := range ModuleExchanges {
		assert.False(t, seen[ex], "duplicate exchange: %s", ex)
		seen[ex] = true
	}

	for _, exp := range expected {
		assert.True(t, seen[exp], "missing exchange: %s", exp)
	}
}
