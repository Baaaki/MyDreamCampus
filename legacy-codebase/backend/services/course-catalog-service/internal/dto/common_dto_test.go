package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaginationRequest_SetDefaults(t *testing.T) {
	tests := []struct {
		name      string
		page, lim int
		wantPage  int
		wantLim   int
	}{
		{"both zero", 0, 0, 1, 20},
		{"page zero, limit set", 0, 50, 1, 50},
		{"page set, limit zero", 3, 0, 3, 20},
		{"both set", 5, 100, 5, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PaginationRequest{Page: tt.page, Limit: tt.lim}
			p.SetDefaults()
			assert.Equal(t, tt.wantPage, p.Page)
			assert.Equal(t, tt.wantLim, p.Limit)
		})
	}
}

func TestPaginationRequest_GetOffset(t *testing.T) {
	cases := []struct {
		page, limit, offset int
	}{
		{1, 20, 0},
		{2, 20, 20},
		{5, 50, 200},
	}
	for _, c := range cases {
		p := PaginationRequest{Page: c.page, Limit: c.limit}
		assert.Equal(t, c.offset, p.GetOffset())
	}
}

func TestNewPaginationResponse(t *testing.T) {
	tests := []struct {
		page, limit, total, expectedPages int
	}{
		{1, 20, 0, 1},     // empty result still gets page 1
		{1, 20, 19, 1},
		{1, 20, 20, 1},
		{1, 20, 21, 2},
		{2, 50, 100, 2},
		{1, 10, 95, 10},   // ceiling division
	}
	for _, tt := range tests {
		r := NewPaginationResponse(tt.page, tt.limit, tt.total)
		assert.Equal(t, tt.page, r.Page)
		assert.Equal(t, tt.limit, r.Limit)
		assert.Equal(t, tt.total, r.Total)
		assert.Equal(t, tt.expectedPages, r.TotalPages,
			"page=%d limit=%d total=%d", tt.page, tt.limit, tt.total)
	}
}
