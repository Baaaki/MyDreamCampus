package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttendanceThreshold_TheoryScaling(t *testing.T) {
	tests := []struct {
		name          string
		totalSessions int64
		expectedMin   int
	}{
		{name: "standard 14 weeks", totalSessions: 14, expectedMin: 10},
		{name: "half semester 7 weeks", totalSessions: 7, expectedMin: 5},
		{name: "short course 8 sessions", totalSessions: 8, expectedMin: 6},
		{name: "single session", totalSessions: 1, expectedMin: 1},
		{name: "zero sessions", totalSessions: 0, expectedMin: 0},
		{name: "negative sessions safe fallback", totalSessions: -5, expectedMin: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := minTheoryRequired(tt.totalSessions)
			assert.Equal(t, tt.expectedMin, got)
		})
	}
}

func TestAttendanceThreshold_LabScaling(t *testing.T) {
	tests := []struct {
		name          string
		totalSessions int64
		expectedMin   int
	}{
		{name: "standard 14 weeks", totalSessions: 14, expectedMin: 11},
		{name: "half semester 7 weeks", totalSessions: 7, expectedMin: 6},
		{name: "extended semester 16 weeks", totalSessions: 16, expectedMin: 13},
		{name: "single session", totalSessions: 1, expectedMin: 1},
		{name: "zero sessions", totalSessions: 0, expectedMin: 0},
		{name: "negative sessions safe fallback", totalSessions: -1, expectedMin: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := minLabRequired(tt.totalSessions)
			assert.Equal(t, tt.expectedMin, got)
		})
	}
}
