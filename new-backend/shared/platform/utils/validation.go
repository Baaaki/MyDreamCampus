package utils

import "regexp"

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail checks if email format is valid
func ValidateEmail(email string) bool {
	if len(email) < 3 || len(email) > 255 {
		return false
	}
	return emailRegex.MatchString(email)
}

// ValidateStudentNumber checks student number format
// Expected format: YYYYNNNNNNN (year + 7 digits)
func ValidateStudentNumber(studentNumber string) bool {
	if len(studentNumber) < 7 || len(studentNumber) > 50 {
		return false
	}
	return true
}
