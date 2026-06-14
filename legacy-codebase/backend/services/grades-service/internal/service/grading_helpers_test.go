package service

import (
	"math"
	"testing"

	"github.com/baaaki/mydreamcampus/grades-service/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestDetermineGradingType(t *testing.T) {
	cases := []struct {
		mean float64
		want db.GradingTypeEnum
	}{
		{0.0, db.GradingTypeEnumRelative},
		{59.99, db.GradingTypeEnumRelative},
		{60.0, db.GradingTypeEnumAbsolute},
		{75.5, db.GradingTypeEnumAbsolute},
		{100.0, db.GradingTypeEnumAbsolute},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, determineGradingType(c.mean), "mean=%.2f", c.mean)
	}
}

func TestCalculateAbsoluteGradePoint(t *testing.T) {
	cases := []struct {
		avg  float64
		want db.GradePointEnum
	}{
		{100.0, db.GradePointEnum400}, // AA
		{90.0, db.GradePointEnum400},
		{89.99, db.GradePointEnum375}, // BA+
		{87.5, db.GradePointEnum375},
		{85.0, db.GradePointEnum350},
		{82.5, db.GradePointEnum325},
		{80.0, db.GradePointEnum300}, // BB
		{77.5, db.GradePointEnum275},
		{75.0, db.GradePointEnum250},
		{72.5, db.GradePointEnum225},
		{70.0, db.GradePointEnum200}, // CC
		{67.5, db.GradePointEnum175},
		{65.0, db.GradePointEnum150},
		{62.5, db.GradePointEnum125},
		{60.0, db.GradePointEnum100}, // DD (passing minimum)
		{55.0, db.GradePointEnum050}, // FD
		{49.99, db.GradePointEnum000}, // FF
		{0.0, db.GradePointEnum000},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, calculateAbsoluteGradePoint(c.avg),
			"avg=%.2f", c.avg)
	}
}

func TestCalculateZScoreGradePoint(t *testing.T) {
	t.Run("zero stddev gives default CC", func(t *testing.T) {
		gp, z := calculateZScoreGradePoint(50, 50, 0)
		assert.Equal(t, db.GradePointEnum200, gp)
		assert.Equal(t, 0.0, z)
	})

	t.Run("z=2 gets AA", func(t *testing.T) {
		gp, z := calculateZScoreGradePoint(70, 50, 10)
		assert.Equal(t, db.GradePointEnum400, gp)
		assert.InDelta(t, 2.0, z, 0.001)
	})

	t.Run("z=-2 gets FF", func(t *testing.T) {
		gp, z := calculateZScoreGradePoint(30, 50, 10)
		assert.Equal(t, db.GradePointEnum000, gp)
		assert.InDelta(t, -2.0, z, 0.001)
	})

	t.Run("at-mean gets CC", func(t *testing.T) {
		gp, z := calculateZScoreGradePoint(50, 50, 10)
		assert.Equal(t, db.GradePointEnum200, gp)
		assert.Equal(t, 0.0, z)
	})

	t.Run("z=-1 gets DD passing minimum", func(t *testing.T) {
		gp, _ := calculateZScoreGradePoint(40, 50, 10)
		assert.Equal(t, db.GradePointEnum100, gp)
	})
}

func TestIsPassing(t *testing.T) {
	passing := []db.GradePointEnum{
		db.GradePointEnum400, db.GradePointEnum375, db.GradePointEnum350,
		db.GradePointEnum325, db.GradePointEnum300, db.GradePointEnum275,
		db.GradePointEnum250, db.GradePointEnum225, db.GradePointEnum200,
		db.GradePointEnum175, db.GradePointEnum150, db.GradePointEnum125,
		db.GradePointEnum100, // DD is minimum passing
	}
	for _, gp := range passing {
		assert.True(t, isPassing(gp), "%v must be passing", gp)
	}

	failing := []db.GradePointEnum{
		db.GradePointEnum050, // FD
		db.GradePointEnum000, // FF
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
