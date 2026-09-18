package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"first.last+tag@sub.domain.tr", true},
		{"x@y.io", true},
		{"", false},
		{"no-at-sign", false},
		{"@no-local.com", false},
		{"local@", false},
		{"local@domain", false},                            // missing TLD
		{"a@b.c", false},                                   // tld too short
		{strings.Repeat("a", 250) + "@example.com", false}, // > 255
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			assert.Equal(t, tt.want, ValidateEmail(tt.email))
		})
	}
}

func TestValidateStudentNumber(t *testing.T) {
	assert.True(t, ValidateStudentNumber("2024001234"))
	assert.True(t, ValidateStudentNumber("1234567"))
	assert.False(t, ValidateStudentNumber("12345"))                 // too short
	assert.False(t, ValidateStudentNumber(strings.Repeat("1", 60))) // > 50
	assert.False(t, ValidateStudentNumber(""))
}
