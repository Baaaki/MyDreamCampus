package service

import (
	"math"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/db"
	"github.com/stretchr/testify/assert"
)

func TestDetermineGradingType(t *testing.T) {
	cases := []struct {
		mean float64
		want db.GradesGradingTypeEnum
	}{
		{0.0, db.GradesGradingTypeEnumRelative},
		{59.99, db.GradesGradingTypeEnumRelative},
		{60.0, db.GradesGradingTypeEnumAbsolute},
		{75.5, db.GradesGradingTypeEnumAbsolute},
		{100.0, db.GradesGradingTypeEnumAbsolute},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, determineGradingType(c.mean), "mean=%.2f", c.mean)
	}
}

func TestCalculateAbsoluteGradePoint(t *testing.T) {
	cases := []struct {
		avg  float64
		want db.GradesGradePointEnum
	}{
		{100.0, db.GradesGradePointEnum400}, // AA
		{90.0, db.GradesGradePointEnum400},
		{89.99, db.GradesGradePointEnum375}, // BA+
		{87.5, db.GradesGradePointEnum375},
		{85.0, db.GradesGradePointEnum350},
		{82.5, db.GradesGradePointEnum325},
		{80.0, db.GradesGradePointEnum300}, // BB
		{77.5, db.GradesGradePointEnum275},
		{75.0, db.GradesGradePointEnum250},
		{72.5, db.GradesGradePointEnum225},
		{70.0, db.GradesGradePointEnum200}, // CC
		{67.5, db.GradesGradePointEnum175},
		{65.0, db.GradesGradePointEnum150},
		{62.5, db.GradesGradePointEnum125},
		{60.0, db.GradesGradePointEnum100}, // DD (passing minimum)
		{55.0, db.GradesGradePointEnum050}, // FD
		{49.99, db.GradesGradePointEnum000}, // FF
		{0.0, db.GradesGradePointEnum000},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, calculateAbsoluteGradePoint(c.avg),
			"avg=%.2f", c.avg)
	}
}

func TestCalculateZScoreGradePoint(t *testing.T) {
	t.Run("zero stddev gives default CC", func(t *testing.T) {
		gp, z := calculateZScoreGradePoint(50, 50, 0)
		assert.Equal(t, db.GradesGradePointEnum200, gp)
		assert.Equal(t, 0.0, z)
	})

	t.Run("z=2 gets AA", func(t *testing.T) {
		gp, z := calculateZScoreGradePoint(70, 50, 10)
		assert.Equal(t, db.GradesGradePointEnum400, gp)
		assert.InDelta(t, 2.0, z, 0.001)
	})

	t.Run("z=-2 gets FF", func(t *testing.T) {
		gp, z := calculateZScoreGradePoint(30, 50, 10)
		assert.Equal(t, db.GradesGradePointEnum000, gp)
		assert.InDelta(t, -2.0, z, 0.001)
	})

	t.Run("at-mean gets CC", func(t *testing.T) {
		gp, z := calculateZScoreGradePoint(50, 50, 10)
		assert.Equal(t, db.GradesGradePointEnum200, gp)
		assert.Equal(t, 0.0, z)
	})

	t.Run("z=-1 gets DD passing minimum", func(t *testing.T) {
		gp, _ := calculateZScoreGradePoint(40, 50, 10)
		assert.Equal(t, db.GradesGradePointEnum100, gp)
	})
}

func TestIsPassing(t *testing.T) {
	passing := []db.GradesGradePointEnum{
		db.GradesGradePointEnum400, db.GradesGradePointEnum375, db.GradesGradePointEnum350,
		db.GradesGradePointEnum325, db.GradesGradePointEnum300, db.GradesGradePointEnum275,
		db.GradesGradePointEnum250, db.GradesGradePointEnum225, db.GradesGradePointEnum200,
		db.GradesGradePointEnum175, db.GradesGradePointEnum150, db.GradesGradePointEnum125,
		db.GradesGradePointEnum100, // DD is minimum passing
	}
	for _, gp := range passing {
		assert.True(t, isPassing(gp), "%v must be passing", gp)
	}

	failing := []db.GradesGradePointEnum{
		db.GradesGradePointEnum050, // FD
		db.GradesGradePointEnum000, // FF
	}
	for _, gp := range failing {
		assert.False(t, isPassing(gp), "%v must NOT be passing", gp)
	}
}

func TestCalculateClassStatistics(t *testing.T) {
	t.Run("empty input returns zero", func(t *testing.T) {
		s := calculateClassStatistics([]float64{})
		assert.Equal(t, ClassStatistics{}, s)
	})

	t.Run("single score: mean=score, stddev=0", func(t *testing.T) {
		s := calculateClassStatistics([]float64{75})
		assert.Equal(t, 75.0, s.Mean)
		assert.Equal(t, 0.0, s.StdDev)
		assert.Equal(t, 75.0, s.Min)
		assert.Equal(t, 75.0, s.Max)
		assert.Equal(t, 1, s.Count)
	})

	t.Run("uniform scores", func(t *testing.T) {
		s := calculateClassStatistics([]float64{60, 60, 60, 60})
		assert.Equal(t, 60.0, s.Mean)
		assert.Equal(t, 0.0, s.StdDev)
	})

	t.Run("normal distribution roughly", func(t *testing.T) {
		s := calculateClassStatistics([]float64{50, 60, 70, 80, 90})
		assert.Equal(t, 70.0, s.Mean)
		assert.InDelta(t, math.Sqrt(200), s.StdDev, 0.01) // population stddev
		assert.Equal(t, 50.0, s.Min)
		assert.Equal(t, 90.0, s.Max)
		assert.Equal(t, 5, s.Count)
	})

	t.Run("rounds to 2 decimal places", func(t *testing.T) {
		s := calculateClassStatistics([]float64{1.0/3, 2.0/3, 1.0})
		// Just verify it's rounded — actual value matters less than precision
		mean := s.Mean
		assert.Equal(t, math.Round(mean*100)/100, mean,
			"mean must be 2-decimal rounded")
	})
}

func TestCalculateWeightedAverage(t *testing.T) {
	schema := []AssessmentSchemaItem{
		{Slug: "midterm", Weight: 40},
		{Slug: "final", Weight: 60},
	}

	t.Run("standard weighted average", func(t *testing.T) {
		scores := map[string]float64{"midterm": 80, "final": 70}
		// (80*40 + 70*60) / 100 = 7400/100 = 74.0
		assert.Equal(t, 74.0, calculateWeightedAverage(scores, schema))
	})

	t.Run("partial scores ignore missing weights", func(t *testing.T) {
		scores := map[string]float64{"midterm": 80}
		// 80*40 / 40 = 80.0
		assert.Equal(t, 80.0, calculateWeightedAverage(scores, schema))
	})

	t.Run("no scores returns 0", func(t *testing.T) {
		assert.Equal(t, 0.0, calculateWeightedAverage(map[string]float64{}, schema))
	})

	t.Run("empty schema returns 0", func(t *testing.T) {
		assert.Equal(t, 0.0, calculateWeightedAverage(map[string]float64{"x": 90}, nil))
	})

	t.Run("equal weights", func(t *testing.T) {
		eqSchema := []AssessmentSchemaItem{
			{Slug: "q1", Weight: 25}, {Slug: "q2", Weight: 25},
			{Slug: "midterm", Weight: 25}, {Slug: "final", Weight: 25},
		}
		scores := map[string]float64{"q1": 100, "q2": 80, "midterm": 60, "final": 40}
		assert.Equal(t, 70.0, calculateWeightedAverage(scores, eqSchema))
	})

	t.Run("rounds to 2 decimals", func(t *testing.T) {
		eqSchema := []AssessmentSchemaItem{{Slug: "x", Weight: 100}}
		scores := map[string]float64{"x": 33.333333}
		got := calculateWeightedAverage(scores, eqSchema)
		assert.Equal(t, 33.33, got)
	})
}
