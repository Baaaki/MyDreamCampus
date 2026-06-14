package service

import (
	"math"
	"slices"

	"github.com/baaaki/mydreamcampus/grades-service/internal/db"
)

// Grading type decision based on class mean
func determineGradingType(classMean float64) db.GradingTypeEnum {
	if classMean >= 60.0 {
		return db.GradingTypeEnumAbsolute
	}
	return db.GradingTypeEnumRelative
}

// Absolute grading (when class mean >= 60)
func calculateAbsoluteGradePoint(average float64) db.GradePointEnum {
	switch {
	case average >= 90.0:
		return db.GradePointEnum400
	case average >= 87.5:
		return db.GradePointEnum375
	case average >= 85.0:
		return db.GradePointEnum350
	case average >= 82.5:
		return db.GradePointEnum325
	case average >= 80.0:
		return db.GradePointEnum300
	case average >= 77.5:
		return db.GradePointEnum275
	case average >= 75.0:
		return db.GradePointEnum250
	case average >= 72.5:
		return db.GradePointEnum225
	case average >= 70.0:
		return db.GradePointEnum200
	case average >= 67.5:
		return db.GradePointEnum175
	case average >= 65.0:
		return db.GradePointEnum150
	case average >= 62.5:
		return db.GradePointEnum125
	case average >= 60.0:
		return db.GradePointEnum100
	case average >= 50.0:
		return db.GradePointEnum050
	default:
		return db.GradePointEnum000
	}
}

// Relative grading (Z-score when class mean < 60)
func calculateZScoreGradePoint(average, mean, stddev float64) (db.GradePointEnum, float64) {
	// If stddev is 0, everyone has the same score
	if stddev == 0 {
		return db.GradePointEnum200, 0.0 // CC
	}

	zScore := (average - mean) / stddev

	var gradePoint db.GradePointEnum
	switch {
	case zScore >= 2.00:
		gradePoint = db.GradePointEnum400
	case zScore >= 1.75:
		gradePoint = db.GradePointEnum375
	case zScore >= 1.50:
		gradePoint = db.GradePointEnum350
	case zScore >= 1.25:
		gradePoint = db.GradePointEnum325
	case zScore >= 1.00:
		gradePoint = db.GradePointEnum300
	case zScore >= 0.75:
		gradePoint = db.GradePointEnum275
	case zScore >= 0.50:
		gradePoint = db.GradePointEnum250
	case zScore >= 0.25:
		gradePoint = db.GradePointEnum225
	case zScore >= 0.00:
		gradePoint = db.GradePointEnum200
	case zScore >= -0.25:
		gradePoint = db.GradePointEnum175
	case zScore >= -0.50:
		gradePoint = db.GradePointEnum150
	case zScore >= -0.75:
		gradePoint = db.GradePointEnum125
	case zScore >= -1.00:
		gradePoint = db.GradePointEnum100
	case zScore >= -1.50:
		gradePoint = db.GradePointEnum050
	default:
		gradePoint = db.GradePointEnum000
	}

	return gradePoint, zScore
}

// Check if grade point is passing (DD and above)
func isPassing(gp db.GradePointEnum) bool {
	passingGrades := []db.GradePointEnum{
		db.GradePointEnum400, db.GradePointEnum375, db.GradePointEnum350,
		db.GradePointEnum325, db.GradePointEnum300, db.GradePointEnum275,
		db.GradePointEnum250, db.GradePointEnum225, db.GradePointEnum200,
		db.GradePointEnum175, db.GradePointEnum150, db.GradePointEnum125,
		db.GradePointEnum100, // DD is the minimum passing grade
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
