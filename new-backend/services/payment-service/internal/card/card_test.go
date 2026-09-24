package card

import (
	"testing"
	"time"

	serviceErrors "github.com/baaaki/mydreamcampus/payment/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var now = time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

func validInput(number string) Input {
	return Input{Number: number, ExpMonth: 12, ExpYear: 30, CVC: "123", Holder: "Zeynep Şahin"}
}

func TestCheck_TestCards_ReturnPublishedOutcome(t *testing.T) {
	cases := map[string]string{
		"4242 4242 4242 4242": "",
		"4000 0000 0000 0002": "Kart reddedildi",
		"4000-0000-0000-9995": "Yetersiz bakiye",
		"5555555555554444":    "", // any other Luhn-valid number succeeds
	}
	for number, reason := range cases {
		t.Run(number, func(t *testing.T) {
			res, err := Check(validInput(number), now)

			require.NoError(t, err)
			assert.Equal(t, reason, res.DeclineReason)
		})
	}
}

func TestCheck_Brands_ReadFromPrefix(t *testing.T) {
	cases := map[string]string{
		"4242424242424242": "Visa",
		"5555555555554444": "Mastercard",
		"2223003122003222": "Mastercard",
		"378282246310005":  "Amex",
		"9792030000000000": "Troy",
		"6011111111111117": "",
	}
	for number, want := range cases {
		t.Run(number, func(t *testing.T) {
			res, err := Check(validInput(number), now)

			require.NoError(t, err)
			assert.Equal(t, want, res.Brand)
			assert.Equal(t, number[len(number)-4:], res.Last4)
		})
	}
}

func TestCheck_InvalidForm_ReturnsValidationError(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Input)
		want   error
	}{
		{"luhn fails", func(in *Input) { in.Number = "4242424242424241" }, serviceErrors.ErrInvalidCardNumber},
		{"letters in number", func(in *Input) { in.Number = "4242abcd42424242" }, serviceErrors.ErrInvalidCardNumber},
		{"too short", func(in *Input) { in.Number = "42424242424" }, serviceErrors.ErrInvalidCardNumber},
		{"empty number", func(in *Input) { in.Number = "" }, serviceErrors.ErrInvalidCardNumber},
		{"month 13", func(in *Input) { in.ExpMonth = 13 }, serviceErrors.ErrInvalidExpiry},
		{"month 0", func(in *Input) { in.ExpMonth = 0 }, serviceErrors.ErrInvalidExpiry},
		{"year too far", func(in *Input) { in.ExpYear = 2099 }, serviceErrors.ErrInvalidExpiry},
		{"expired last month", func(in *Input) { in.ExpMonth, in.ExpYear = 8, 26 }, serviceErrors.ErrCardExpired},
		{"cvc two digits", func(in *Input) { in.CVC = "12" }, serviceErrors.ErrInvalidCVC},
		{"cvc five digits", func(in *Input) { in.CVC = "12345" }, serviceErrors.ErrInvalidCVC},
		{"cvc not digits", func(in *Input) { in.CVC = "12a" }, serviceErrors.ErrInvalidCVC},
		{"blank holder", func(in *Input) { in.Holder = "   " }, serviceErrors.ErrInvalidCardholder},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput("4242424242424242")
			tc.mutate(&in)

			_, err := Check(in, now)

			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestCheck_ExpiryMonth_ValidThroughItsLastDay(t *testing.T) {
	in := validInput("4242424242424242")
	in.ExpMonth, in.ExpYear = 9, 2026

	_, err := Check(in, time.Date(2026, 9, 30, 23, 59, 0, 0, time.UTC))
	require.NoError(t, err)

	_, err = Check(in, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	assert.ErrorIs(t, err, serviceErrors.ErrCardExpired)
}

func TestCheck_FourDigitCVC_Accepted(t *testing.T) {
	in := validInput("378282246310005")
	in.CVC = "1234"

	_, err := Check(in, now)

	assert.NoError(t, err)
}
