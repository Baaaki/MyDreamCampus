package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQRWindow_Calculation(t *testing.T) {
	rotation := 15 * time.Second

	tests := []struct {
		name     string
		t        time.Time
		expected int64
	}{
		{"epoch", time.Unix(0, 0), 0},
		{"14 seconds (same window 0)", time.Unix(14, 0), 0},
		{"15 seconds (window 1 tick)", time.Unix(15, 0), 1},
		{"29 seconds (window 1)", time.Unix(29, 0), 1},
		{"30 seconds (window 2 tick)", time.Unix(30, 0), 2},
		{"fixed instant", time.Unix(1500, 0), 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, QRWindow(tt.t, rotation))
		})
	}
}
