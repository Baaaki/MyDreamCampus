package utils

import (
	"regexp"
	"unicode"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail checks if email format is valid
func ValidateEmail(email string) bool {
	if len(email) < 3 || len(email) > 255 {
		return false
	}
	return emailRegex.MatchString(email)
}

// ValidatePassword checks password strength
// Requirements:
// - Length between 12 and 128 characters
// - At least one uppercase letter
// - At least one lowercase letter
// - At least one digit
// - At least one special character (non-alphanumeric)
func ValidatePassword(password string) bool {
	if len(password) < 12 || len(password) > 128 {
		return false
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasDigit   = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// ValidateStudentNumber checks student number format
// Expected format: YYYYNNNNNNN (year + 7 digits)
func ValidateStudentNumber(studentNumber string) bool {
	if len(studentNumber) < 7 || len(studentNumber) > 50 {
		return false
	}
	return true
}
