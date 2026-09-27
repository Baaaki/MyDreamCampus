package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidPeriodType(t *testing.T) {
	tests := []struct {
		periodType string
		expected   bool
	}{
		{PeriodTypeCatalog, true},
		{PeriodTypeEnrollment, true},
		{PeriodTypeGrading, true},
		{PeriodTypeAttendance, true},
		{"", false},
		{"custom", false},
		{"CATALOG", false},
		{"exams", false},
	}

	for _, tt := range tests {
		t.Run(tt.periodType, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsValidPeriodType(tt.periodType))
		})
	}
}

func TestProjectedPeriodTypes(t *testing.T) {
	assert.Len(t, ProjectedPeriodTypes, 3)
	assert.Contains(t, ProjectedPeriodTypes, PeriodTypeEnrollment)
	assert.Contains(t, ProjectedPeriodTypes, PeriodTypeGrading)
	assert.Contains(t, ProjectedPeriodTypes, PeriodTypeAttendance)
	assert.NotContains(t, ProjectedPeriodTypes, PeriodTypeCatalog)
}

func TestNewSimplePeriodRepository(t *testing.T) {
	t.Run("panics on invalid schema", func(t *testing.T) {
		assert.PanicsWithValue(t, "invalid period schema: invalid_schema", func() {
			NewSimplePeriodRepository(nil, "invalid_schema")
		})
		assert.PanicsWithValue(t, "invalid period schema: public", func() {
			NewSimplePeriodRepository(nil, "public")
		})
		assert.PanicsWithValue(t, "invalid period schema: ", func() {
			NewSimplePeriodRepository(nil, "")
		})
	})

	t.Run("accepts allowed schemas", func(t *testing.T) {
		schemas := []string{"course_catalog", "enrollment", "attendance", "grades"}
		for _, schema := range schemas {
			repo := NewSimplePeriodRepository(nil, schema)
			require.NotNil(t, repo)
			assert.Equal(t, schema+".academic_periods", repo.Table())
			if schema == "course_catalog" {
				assert.True(t, repo.HasPeriodType())
			} else {
				assert.False(t, repo.HasPeriodType())
			}
			assert.Nil(t, repo.Pool())
		}
	})
}

func TestSimplePeriodRepository_RequireCatalog(t *testing.T) {
	catalogRepo := NewSimplePeriodRepository(nil, "course_catalog")
	assert.NoError(t, catalogRepo.requireCatalog("testOp"))

	enrollmentRepo := NewSimplePeriodRepository(nil, "enrollment")
	err := enrollmentRepo.requireCatalog("testOp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "testOp requires the catalog period table")
}

func TestSimplePeriodRepository_Scope(t *testing.T) {
	t.Run("catalog repo with period type appends condition", func(t *testing.T) {
		repo := NewSimplePeriodRepository(nil, "course_catalog")
		args := []any{"arg1"}
		clause, newArgs := repo.scope(PeriodTypeEnrollment, args)
		assert.Equal(t, " AND period_type = $2", clause)
		assert.Equal(t, []any{"arg1", PeriodTypeEnrollment}, newArgs)
	})

	t.Run("catalog repo with empty period type returns unchanged", func(t *testing.T) {
		repo := NewSimplePeriodRepository(nil, "course_catalog")
		args := []any{"arg1"}
		clause, newArgs := repo.scope("", args)
		assert.Empty(t, clause)
		assert.Equal(t, args, newArgs)
	})

	t.Run("non-catalog repo returns unchanged even with period type", func(t *testing.T) {
		repo := NewSimplePeriodRepository(nil, "attendance")
		args := []any{"arg1", "arg2"}
		clause, newArgs := repo.scope(PeriodTypeAttendance, args)
		assert.Empty(t, clause)
		assert.Equal(t, args, newArgs)
	})
}
