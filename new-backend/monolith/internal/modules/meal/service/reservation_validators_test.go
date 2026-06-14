package service

import (
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/meal/errors"
	"github.com/stretchr/testify/assert"
)

// These tests exercise the pure validators on ReservationService — the parts
// that do NOT touch a repository. They reuse the newTestService / defaultCfg
// / freezeAt helpers from reservation_service_helpers_test.go.

func TestValidateMealTimeAndMenu(t *testing.T) {
	s := newTestService(defaultCfg())

	cafFull := db.Cafeteria{ServesDinner: true, HasVeganMenu: true}
	cafLunchOnly := db.Cafeteria{ServesDinner: false, HasVeganMenu: true}
	cafNoVegan := db.Cafeteria{ServesDinner: true, HasVeganMenu: false}

	t.Run("lunch + normal", func(t *testing.T) {
		assert.NoError(t, s.validateMealTimeAndMenu("lunch", "normal", cafFull))
	})

	t.Run("dinner + vegan when cafeteria supports both", func(t *testing.T) {
		assert.NoError(t, s.validateMealTimeAndMenu("dinner", "vegan", cafFull))
	})

	t.Run("invalid meal time", func(t *testing.T) {
		err := s.validateMealTimeAndMenu("brunch", "normal", cafFull)
		assert.ErrorIs(t, err, serviceErrors.ErrInvalidMealTime,
			"only lunch/dinner are accepted — quietly accepting 'brunch' would let typos through to the DB")
	})

	t.Run("invalid menu type", func(t *testing.T) {
		err := s.validateMealTimeAndMenu("lunch", "keto", cafFull)
		assert.ErrorIs(t, err, serviceErrors.ErrInvalidMenuType)
	})

	t.Run("dinner refused when cafeteria does not serve dinner", func(t *testing.T) {
		err := s.validateMealTimeAndMenu("dinner", "normal", cafLunchOnly)
		assert.ErrorIs(t, err, serviceErrors.ErrCafeteriaNoDinner,
			"per-cafeteria capability flags must be enforced — bypassing this check books reservations the kitchen cannot fulfil")
	})

	t.Run("vegan refused when cafeteria has no vegan menu", func(t *testing.T) {
		err := s.validateMealTimeAndMenu("lunch", "vegan", cafNoVegan)
		assert.ErrorIs(t, err, serviceErrors.ErrCafeteriaNoVegan)
	})

	t.Run("error order: meal time before menu type", func(t *testing.T) {
		// If both are wrong we surface meal time first — UX expects to fix
		// the more obvious field first. Locking the order.
		err := s.validateMealTimeAndMenu("brunch", "keto", cafFull)
		assert.ErrorIs(t, err, serviceErrors.ErrInvalidMealTime)
	})
}

func TestValidateMealTimeWindow(t *testing.T) {
	s := newTestService(defaultCfg())

	t.Run("inside lunch window", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 28, 12, 0)
		assert.NoError(t, s.validateMealTimeWindow("lunch"))
	})

	t.Run("at lunch start boundary", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 28, 11, 0)
		assert.NoError(t, s.validateMealTimeWindow("lunch"),
			"at the start hour the window is OPEN — half-open interval [start, end)")
	})

	t.Run("at lunch end boundary", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 28, 13, 0)
		err := s.validateMealTimeWindow("lunch")
		assert.ErrorIs(t, err, serviceErrors.ErrOutsideMealTimeWindow,
			"at the end hour the window is CLOSED — booking the closing minute would let students hold up service")
	})

	t.Run("before lunch", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 28, 9, 0)
		err := s.validateMealTimeWindow("lunch")
		assert.ErrorIs(t, err, serviceErrors.ErrOutsideMealTimeWindow)
	})

	t.Run("inside dinner window", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 28, 17, 30)
		assert.NoError(t, s.validateMealTimeWindow("dinner"))
	})

	t.Run("after dinner", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 28, 20, 0)
		err := s.validateMealTimeWindow("dinner")
		assert.ErrorIs(t, err, serviceErrors.ErrOutsideMealTimeWindow)
	})
}

func TestValidateCancelCutoff(t *testing.T) {
	s := newTestService(defaultCfg())
	// CancelCutoffHours = 2; LunchStartHour = 11. So for lunch on day D, the
	// cutoff is D 09:00 UTC+3.
	day := time.Date(2026, time.April, 28, 0, 0, 0, 0, utcPlus3)

	t.Run("well before cutoff — cancellation allowed", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 28, 7, 0)
		assert.NoError(t, s.validateCancelCutoff(day, db.MealMealTimeEnumLunch))
	})

	t.Run("right at the cutoff is still allowed (After is strict)", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 28, 9, 0)
		assert.NoError(t, s.validateCancelCutoff(day, db.MealMealTimeEnumLunch),
			"at exactly cutoff time After() is false; cancellation is allowed. Locking the boundary.")
	})

	t.Run("one minute past cutoff — denied", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 28, 9, 1)
		err := s.validateCancelCutoff(day, db.MealMealTimeEnumLunch)
		assert.ErrorIs(t, err, serviceErrors.ErrCancelCutoffPassed,
			"one minute over the cutoff must reject — kitchen has already prepped portions based on counts")
	})

	t.Run("dinner cutoff uses dinner start hour", func(t *testing.T) {
		// DinnerStartHour = 16, cutoff = 14:00.
		freezeAt(t, 2026, time.April, 28, 13, 59)
		assert.NoError(t, s.validateCancelCutoff(day, db.MealMealTimeEnumDinner))

		freezeAt(t, 2026, time.April, 28, 14, 1)
		err := s.validateCancelCutoff(day, db.MealMealTimeEnumDinner)
		assert.ErrorIs(t, err, serviceErrors.ErrCancelCutoffPassed)
	})

	t.Run("invalid meal time short-circuits", func(t *testing.T) {
		freezeAt(t, 2026, time.April, 28, 0, 0)
		err := s.validateCancelCutoff(day, db.MealTimeEnum("brunch"))
		assert.ErrorIs(t, err, serviceErrors.ErrInvalidMealTime,
			"unknown meal type must surface the meal-time error, NOT silently allow the cancel")
	})
}
