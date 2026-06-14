package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// isValidSemesterFormat is the gatekeeper for the semester string that flows
// into every cross-service event (course.semester.created, enrollment,
// grades). Pinning the exact accepted shape so a loosening — e.g. allowing
// 2025-2027 or "Summer" — surfaces here, not in a downstream consumer.
func TestIsValidSemesterFormat_Accepted(t *testing.T) {
	for _, in := range []string{
		"2025-2026-Fall",
		"2025-2026-Spring",
		"2099-2100-Fall",
		"2000-2001-Spring",
	} {
		t.Run(in, func(t *testing.T) {
			assert.True(t, isValidSemesterFormat(in), "expected %q to be accepted", in)
		})
	}
}

func TestIsValidSemesterFormat_Rejected(t *testing.T) {
	cases := []struct {
		in     string
		reason string
	}{
		{"2025-2025-Fall", "second year must be exactly +1"},
		{"2025-2027-Fall", "gap larger than 1 forbidden"},
		{"2025-2024-Fall", "second year cannot be earlier than first"},
		{"2025-2026-Summer", "only Fall and Spring are allowed"},
		{"2025-2026-fall", "case sensitive — lowercase rejected"},
		{"25-26-Fall", "years must be 4 digits"},
		{"2025/2026/Fall", "separator must be hyphen"},
		{"1999-2000-Fall", "start year below 2000 rejected"},
		{"2101-2102-Fall", "start year above 2100 rejected"},
		{"", "empty string rejected"},
		{"  2025-2026-Fall  ", "no whitespace tolerance — caller must trim"},
		{"2025-2026", "missing season part"},
		{"Fall-2025-2026", "wrong field order"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.False(t, isValidSemesterFormat(tc.in), "expected %q to be rejected (%s)", tc.in, tc.reason)
		})
	}
}
