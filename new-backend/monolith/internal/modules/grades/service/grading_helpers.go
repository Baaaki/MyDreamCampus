package service

import (
	"math"
	"slices"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/grades/db"
)

// Grading type decision based on class mean
func determineGradingType(classMean float64) db.GradesGradingTypeEnum {
	if classMean >= 60.0 {
		return db.GradesGradingTypeEnumAbsolute
	}
	return db.GradesGradingTypeEnumRelative
}

// Absolute grading (when class mean >= 60)
func calculateAbsoluteGradePoint(average float64) db.GradesGradePointEnum {
	switch {
	case average >= 90.0:
		return db.GradesGradePointEnum400
	case average >= 87.5:
		return db.GradesGradePointEnum375
	case average >= 85.0:
		return db.GradesGradePointEnum350
	case average >= 82.5:
		return db.GradesGradePointEnum325
	case average >= 80.0:
		return db.GradesGradePointEnum300
	case average >= 77.5:
		return db.GradesGradePointEnum275
	case average >= 75.0:
		return db.GradesGradePointEnum250
	case average >= 72.5:
		return db.GradesGradePointEnum225
	case average >= 70.0:
		return db.GradesGradePointEnum200
	case average >= 67.5:
		return db.GradesGradePointEnum175
	case average >= 65.0:
		return db.GradesGradePointEnum150
	case average >= 62.5:
		return db.GradesGradePointEnum125
	case average >= 60.0:
		return db.GradesGradePointEnum100
	case average >= 50.0:
		return db.GradesGradePointEnum050
	default:
		return db.GradesGradePointEnum000
	}
}

// Relative grading (Z-score when class mean < 60)
func calculateZScoreGradePoint(average, mean, stddev float64) (db.GradesGradePointEnum, float64) {
	// If stddev is 0, everyone has the same score
	if stddev == 0 {
		return db.GradesGradePointEnum200, 0.0 // CC
	}

	zScore := (average - mean) / stddev

	var gradePoint db.GradesGradePointEnum
	switch {
	case zScore >= 2.00:
		gradePoint = db.GradesGradePointEnum400
	case zScore >= 1.75:
		gradePoint = db.GradesGradePointEnum375
	case zScore >= 1.50:
		gradePoint = db.GradesGradePointEnum350
	case zScore >= 1.25:
		gradePoint = db.GradesGradePointEnum325
	case zScore >= 1.00:
		gradePoint = db.GradesGradePointEnum300
	case zScore >= 0.75:
		gradePoint = db.GradesGradePointEnum275
	case zScore >= 0.50:
		gradePoint = db.GradesGradePointEnum250
	case zScore >= 0.25:
		gradePoint = db.GradesGradePointEnum225
	case zScore >= 0.00:
		gradePoint = db.GradesGradePointEnum200
	case zScore >= -0.25:
		gradePoint = db.GradesGradePointEnum175
	case zScore >= -0.50:
		gradePoint = db.GradesGradePointEnum150
	case zScore >= -0.75:
		gradePoint = db.GradesGradePointEnum125
	case zScore >= -1.00:
		gradePoint = db.GradesGradePointEnum100
	case zScore >= -1.50:
		gradePoint = db.GradesGradePointEnum050
	default:
		gradePoint = db.GradesGradePointEnum000
	}

	return gradePoint, zScore
}

// Check if grade point is passing (DD and above)
func isPassing(gp db.GradesGradePointEnum) bool {
	passingGrades := []db.GradesGradePointEnum{
		db.GradesGradePointEnum400, db.GradesGradePointEnum375, db.GradesGradePointEnum350,
		db.GradesGradePointEnum325, db.GradesGradePointEnum300, db.GradesGradePointEnum275,
		db.GradesGradePointEnum250, db.GradesGradePointEnum225, db.GradesGradePointEnum200,
		db.GradesGradePointEnum175, db.GradesGradePointEnum150, db.GradesGradePointEnum125,
		db.GradesGradePointEnum100, // DD is the minimum passing grade
	}

	return slices.Contains(passingGrades, gp)
}

// Calculate class statistics (mean, stddev, min, max)
type ClassStatistics struct {
	Mean   float64
	StdDev float64
	Min    float64
	Max    float64
	Count  int
}

func calculateClassStatistics(averages []float64) ClassStatistics {
	if len(averages) == 0 {
		return ClassStatistics{}
	}

	// Calculate mean
	sum := 0.0
	min := averages[0]
	max := averages[0]

	for _, avg := range averages {
		sum += avg
		if avg < min {
			min = avg
		}
		if avg > max {
			max = avg
		}
	}

	mean := sum / float64(len(averages))

	// Calculate standard deviation
	variance := 0.0
	for _, avg := range averages {
		variance += math.Pow(avg-mean, 2)
	}
	variance /= float64(len(averages))
	stddev := math.Sqrt(variance)

	return ClassStatistics{
		Mean:   math.Round(mean*100) / 100,
		StdDev: math.Round(stddev*100) / 100,
		Min:    min,
		Max:    max,
		Count:  len(averages),
	}
}

// Calculate weighted average from scores and schema
func calculateWeightedAverage(scores map[string]float64, schema []AssessmentSchemaItem) float64 {
	totalWeighted := 0.0
	totalWeight := 0.0

	for _, item := range schema {
		if score, exists := scores[item.Slug]; exists {
			totalWeighted += score * float64(item.Weight)
			totalWeight += float64(item.Weight)
		}
	}

	if totalWeight == 0 {
		return 0.0
	}

	return math.Round((totalWeighted/totalWeight)*100) / 100
}

// Assessment schema item
type AssessmentSchemaItem struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}
