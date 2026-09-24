// Package card validates the demo checkout's card form and decides its
// outcome from the published test numbers. No card data leaves this
// package except the brand and the last four digits.
package card

import (
	"strings"
	"time"
	"unicode/utf8"

	serviceErrors "github.com/baaaki/mydreamcampus/payment/internal/errors"
)

// Input is the card form as the student submitted it.
type Input struct {
	Number   string
	ExpMonth int
	ExpYear  int
	CVC      string
	Holder   string
}

// Result is what may be kept about a card.
type Result struct {
	Brand string // "" when the prefix matches no known network
	Last4 string
	// DeclineReason is empty when the payment goes through.
	DeclineReason string
}

// Test numbers with a fixed outcome. Every other Luhn-valid number succeeds.
var declines = map[string]string{
	"4000000000000002": "Kart reddedildi",
	"4000000000009995": "Yetersiz bakiye",
}

const maxHolderLength = 100

// Check validates the form against now (the service clock) and returns the
// outcome. A validation error means the form is wrong, not that the
// payment failed.
func Check(in Input, now time.Time) (Result, error) {
	number := normalize(in.Number)
	if len(number) < 12 || len(number) > 19 || !allDigits(number) || !luhnValid(number) {
		return Result{}, serviceErrors.ErrInvalidCardNumber
	}
	if err := checkExpiry(in.ExpMonth, in.ExpYear, now); err != nil {
		return Result{}, err
	}
	if len(in.CVC) < 3 || len(in.CVC) > 4 || !allDigits(in.CVC) {
		return Result{}, serviceErrors.ErrInvalidCVC
	}
	holder := strings.TrimSpace(in.Holder)
	if holder == "" || utf8.RuneCountInString(holder) > maxHolderLength {
		return Result{}, serviceErrors.ErrInvalidCardholder
	}

	return Result{
		Brand:         brand(number),
		Last4:         number[len(number)-4:],
		DeclineReason: declines[number],
	}, nil
}

// normalize drops the grouping the form puts between digit blocks.
func normalize(number string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(number)
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func luhnValid(number string) bool {
	sum := 0
	double := false
	for i := len(number) - 1; i >= 0; i-- {
		d := int(number[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

// checkExpiry accepts a two- or four-digit year. A card is valid through
// the last day of its expiry month.
func checkExpiry(month, year int, now time.Time) error {
	if year < 100 {
		year += 2000
	}
	now = now.UTC()
	if month < 1 || month > 12 || year > now.Year()+20 {
		return serviceErrors.ErrInvalidExpiry
	}
	firstInvalidDay := time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)
	if !now.Before(firstInvalidDay) {
		return serviceErrors.ErrCardExpired
	}
	return nil
}

// brand reads the network from the number's prefix.
func brand(number string) string {
	switch {
	case strings.HasPrefix(number, "9792"):
		return "Troy"
	case strings.HasPrefix(number, "34"), strings.HasPrefix(number, "37"):
		return "Amex"
	case strings.HasPrefix(number, "4"):
		return "Visa"
	case inRange(number[:2], "51", "55"), inRange(number[:4], "2221", "2720"):
		return "Mastercard"
	}
	return ""
}

// inRange compares equal-length digit strings, which order like numbers.
func inRange(prefix, lo, hi string) bool {
	return prefix >= lo && prefix <= hi
}
