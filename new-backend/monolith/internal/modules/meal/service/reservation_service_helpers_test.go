package service

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/config"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// utcPlus3 mirrors the time zone the production code uses for all
// meal-time math. Tests use it so that the wall-clock hour we set with
// clock.Set is the same hour that the function sees.
var utcPlus3 = time.FixedZone("UTC+3", 3*3600)

// newTestService builds a ReservationService wired with only the
// fields the pure helper functions read. Repositories stay nil — the
// tests below must not call any method that touches them.
func newTestService(cfg *config.Config) *ReservationService {
	return &ReservationService{
		cfg:    cfg,
		logger: zap.NewNop(),
	}
}

// defaultCfg returns a config populated with the same defaults the
// production setMealDefaults() applies, so tests behave like prod.
func defaultCfg() *config.Config {
	return &config.Config{
		QR: config.QRConfig{Secret: "test-qr-secret"},
		Reservation: config.ReservationConfig{
			QRValidityWindowSeconds: 30,
			CancelCutoffHours:       2,
		},
		MealTime: config.MealTimeConfig{
			LunchStartHour:  11,
			LunchEndHour:    13,
			DinnerStartHour: 16,
			DinnerEndHour:   19,
		},
	}
}

// freezeAt freezes the package clock at the given UTC+3 wall time and
// returns a cleanup that resets it. Helpers below use it so each test
// case sees a deterministic Now().
func freezeAt(t *testing.T, year int, month time.Month, day, hour, minute int) {
	t.Helper()
	clock.Set(time.Date(year, month, day, hour, minute, 0, 0, utcPlus3))
	t.Cleanup(clock.Reset)
}

func TestReservationService_SignQRPayload(t *testing.T) {
	s := newTestService(defaultCfg())

	t.Run("deterministic for identical inputs", func(t *testing.T) {
		a := s.signQRPayload("payload-x")
		b := s.signQRPayload("payload-x")
		assert.Equal(t, a, b)
	})

	t.Run("differs when payload differs", func(t *testing.T) {
		a := s.signQRPayload("payload-a")
		b := s.signQRPayload("payload-b")
		assert.NotEqual(t, a, b)
	})

	t.Run("differs when secret differs", func(t *testing.T) {
		cfgA := defaultCfg()
		cfgB := defaultCfg()
		cfgB.QR.Secret = "different-secret"

		a := newTestService(cfgA).signQRPayload("p")
		b := newTestService(cfgB).signQRPayload("p")
		assert.NotEqual(t, a, b)
	})

	t.Run("returns 64-char hex (sha256)", func(t *testing.T) {
		sig := s.signQRPayload("payload")
		assert.Len(t, sig, 64)
		_, err := hex.DecodeString(sig)
		assert.NoError(t, err)
	})
}

func TestReservationService_QRWindow(t *testing.T) {
	s := newTestService(defaultCfg()) // 30-second buckets

	t.Run("buckets time by QRValidityWindowSeconds", func(t *testing.T) {
		// 30s window: a known unix value divided by 30 must equal qrWindow.
		t0 := time.Unix(1_700_000_010, 0) // arbitrary fixed instant
		clock.Set(t0)
		t.Cleanup(clock.Reset)
		assert.Equal(t, t0.Unix()/30, s.qrWindow())
	})

	t.Run("stays in same bucket within the window", func(t *testing.T) {
		clock.Set(time.Unix(1_700_000_010, 0))
		first := s.qrWindow()

		clock.Set(time.Unix(1_700_000_010+15, 0)) // still inside same 30s bucket
		t.Cleanup(clock.Reset)
		assert.Equal(t, first, s.qrWindow())
	})

	t.Run("advances to next bucket past the window edge", func(t *testing.T) {
		clock.Set(time.Unix(1_700_000_010, 0))
		first := s.qrWindow()

		clock.Set(time.Unix(1_700_000_010+30, 0)) // crossed the boundary
		t.Cleanup(clock.Reset)
		assert.Equal(t, first+1, s.qrWindow())
	})
}

func TestReservationService_QRRoundTrip(t *testing.T) {
	s := newTestService(defaultCfg())
	clock.Set(time.Unix(1_700_000_010, 0))
	t.Cleanup(clock.Reset)

	t.Run("generate then parse returns the original parts", func(t *testing.T) {
		qr := s.generateQRPayload("cafeA", "2026-04-27", "lunch")

		parts := strings.Split(qr, ":")
		require.Len(t, parts, 5)

		cafeteriaID, date, mealTime, window, signature, err := s.parseQRPayload(qr)
		require.NoError(t, err)
		assert.Equal(t, "cafeA", cafeteriaID)
		assert.Equal(t, "2026-04-27", date)
		assert.Equal(t, "lunch", mealTime)
		assert.Equal(t, fmt.Sprintf("%d", s.qrWindow()), window)
		assert.NotEmpty(t, signature)
	})

	t.Run("parse rejects payloads without exactly five parts", func(t *testing.T) {
		cases := []string{
			"only-one",
			"cafe:date:lunch",
			"a:b:c:d:e:extra",
			"",
		}
		for _, c := range cases {
			_, _, _, _, _, err := s.parseQRPayload(c)
			assert.Error(t, err, "input=%q", c)
		}
	})
}

func TestReservationService_VerifyQRSignature(t *testing.T) {
	s := newTestService(defaultCfg())

	// Pin the clock so the helpers under test agree on what "current bucket" is.
	clock.Set(time.Unix(1_700_000_010, 0))
	t.Cleanup(clock.Reset)

	cafeteriaID, date, mealTime := "cafeA", "2026-04-27", "lunch"
	current := s.qrWindow()

	signFor := func(window int64) string {
		payload := fmt.Sprintf("%s:%s:%s:%d", cafeteriaID, date, mealTime, window)
		return s.signQRPayload(payload)
	}

	t.Run("accepts current bucket signature", func(t *testing.T) {
		assert.True(t, s.verifyQRSignature(
			cafeteriaID, date, mealTime,
			fmt.Sprintf("%d", current),
			signFor(current),
		))
	})

	t.Run("accepts previous bucket (boundary grace)", func(t *testing.T) {
		assert.True(t, s.verifyQRSignature(
			cafeteriaID, date, mealTime,
			fmt.Sprintf("%d", current-1),
			signFor(current-1),
		))
	})

	t.Run("rejects bucket two windows old", func(t *testing.T) {
		assert.False(t, s.verifyQRSignature(
			cafeteriaID, date, mealTime,
			fmt.Sprintf("%d", current-2),
			signFor(current-2),
		))
	})

	t.Run("rejects future bucket", func(t *testing.T) {
		assert.False(t, s.verifyQRSignature(
			cafeteriaID, date, mealTime,
			fmt.Sprintf("%d", current+1),
			signFor(current+1),
		))
	})

	t.Run("rejects tampered signature with valid window", func(t *testing.T) {
		assert.False(t, s.verifyQRSignature(
			cafeteriaID, date, mealTime,
			fmt.Sprintf("%d", current),
			"deadbeef",
		))
	})

	t.Run("rejects non-numeric window", func(t *testing.T) {
		assert.False(t, s.verifyQRSignature(
			cafeteriaID, date, mealTime, "not-a-number", signFor(current),
		))
	})

	t.Run("rejects payload tampering (cafeteria swap)", func(t *testing.T) {
		// Signature was issued for cafeA but the scan reports cafeB —
		// verification must fail even if the window is current.
		assert.False(t, s.verifyQRSignature(
			"cafeB", date, mealTime,
			fmt.Sprintf("%d", current),
			signFor(current),
		))
	})
}

func TestReservationService_ValidateMealTimeWindow(t *testing.T) {
	s := newTestService(defaultCfg()) // lunch [11,13), dinner [16,19)

	cases := []struct {
		name     string
		hour     int
		mealTime string
		wantErr  error
	}{
		{"lunch start hour is allowed", 11, "lunch", nil},
		{"mid-lunch is allowed", 12, "lunch", nil},
		{"lunch end hour is exclusive", 13, "lunch", serviceErrors.ErrOutsideMealTimeWindow},
		{"before lunch rejected", 10, "lunch", serviceErrors.ErrOutsideMealTimeWindow},

		{"dinner start hour is allowed", 16, "dinner", nil},
		{"mid-dinner is allowed", 18, "dinner", nil},
		{"dinner end hour is exclusive", 19, "dinner", serviceErrors.ErrOutsideMealTimeWindow},
		{"before dinner rejected", 15, "dinner", serviceErrors.ErrOutsideMealTimeWindow},

		// validateMealTimeWindow only validates lunch/dinner — anything else is
		// accepted as a no-op. Documenting that contract here so a future
		// refactor that tightens it has to update the test consciously.
		{"unknown meal time is a no-op", 3, "snack", nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			freezeAt(t, 2026, time.April, 27, c.hour, 0)
			err := s.validateMealTimeWindow(c.mealTime)
			if c.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, c.wantErr)
			}
		})
	}
}

func TestReservationService_ValidateCancelCutoff(t *testing.T) {
	cfg := defaultCfg() // CancelCutoffHours=2, lunch=11, dinner=16
	s := newTestService(cfg)
	reservationDate := time.Date(2026, time.April, 27, 0, 0, 0, 0, utcPlus3)

	t.Run("well before cutoff is allowed", func(t *testing.T) {
		// lunch starts at 11 UTC+3, cutoff at 09 UTC+3, so 08:00 is fine.
		freezeAt(t, 2026, time.April, 27, 8, 0)
		assert.NoError(t, s.validateCancelCutoff(reservationDate, db.MealMealTimeEnumLunch))
	})

	t.Run("exactly at cutoff is still allowed (After is strict)", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 27, 9, 0)
		assert.NoError(t, s.validateCancelCutoff(reservationDate, db.MealMealTimeEnumLunch))
	})

	t.Run("one minute past cutoff is rejected", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 27, 9, 1)
		err := s.validateCancelCutoff(reservationDate, db.MealMealTimeEnumLunch)
		assert.ErrorIs(t, err, serviceErrors.ErrCancelCutoffPassed)
	})

	t.Run("dinner uses dinner start hour", func(t *testing.T) {
		// dinner starts 16 UTC+3, cutoff 14 UTC+3.
		freezeAt(t, 2026, time.April, 27, 14, 1)
		err := s.validateCancelCutoff(reservationDate, db.MealMealTimeEnumDinner)
		assert.ErrorIs(t, err, serviceErrors.ErrCancelCutoffPassed)
	})

	t.Run("invalid meal time returns ErrInvalidMealTime", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 27, 8, 0)
		err := s.validateCancelCutoff(reservationDate, db.MealTimeEnum("brunch"))
		assert.ErrorIs(t, err, serviceErrors.ErrInvalidMealTime)
	})
}

func TestReservationService_ParseMealTimeEnum(t *testing.T) {
	s := newTestService(defaultCfg())

	cases := map[string]struct {
		want    db.MealTimeEnum
		wantErr bool
	}{
		"lunch":  {db.MealMealTimeEnumLunch, false},
		"dinner": {db.MealMealTimeEnumDinner, false},
		"":       {"", true},
		"snack":  {"", true},
		"LUNCH":  {"", true}, // case-sensitive by design
	}

	for input, c := range cases {
		t.Run("input="+input, func(t *testing.T) {
			got, err := s.parseMealTimeEnum(input)
			if c.wantErr {
				assert.ErrorIs(t, err, serviceErrors.ErrInvalidMealTime)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.want, got)
			}
		})
	}
}

func TestReservationService_ParseMenuTypeEnum(t *testing.T) {
	s := newTestService(defaultCfg())

	cases := map[string]struct {
		want    db.MenuTypeEnum
		wantErr bool
	}{
		"normal": {db.MealMenuTypeEnumNormal, false},
		"vegan":  {db.MealMenuTypeEnumVegan, false},
		"":       {"", true},
		"halal":  {"", true},
	}

	for input, c := range cases {
		t.Run("input="+input, func(t *testing.T) {
			got, err := s.parseMenuTypeEnum(input)
			if c.wantErr {
				assert.ErrorIs(t, err, serviceErrors.ErrInvalidMenuType)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.want, got)
			}
		})
	}
}

func TestReservationService_ParseStatusEnum(t *testing.T) {
	s := newTestService(defaultCfg())

	cases := map[string]struct {
		want    db.ReservationStatusEnum
		wantErr bool
	}{
		"pending":   {db.MealReservationStatusEnumPending, false},
		"confirmed": {db.MealReservationStatusEnumConfirmed, false},
		"cancelled": {db.MealReservationStatusEnumCancelled, false},
		"expired":   {db.MealReservationStatusEnumExpired, false},
		"":          {"", true},
		"unknown":   {"", true},
	}

	for input, c := range cases {
		t.Run("input="+input, func(t *testing.T) {
			got, err := s.parseStatusEnum(input)
			if c.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.want, got)
			}
		})
	}
}
