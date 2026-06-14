package errors

import (
	"net/http"
	"testing"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/stretchr/testify/assert"
)

func TestMealErrors_HTTPStatuses(t *testing.T) {
	cases := map[string]struct {
		err  *sharedErrors.AppError
		want int
	}{
		"CAFETERIA_NOT_FOUND":         {ErrCafeteriaNotFound, http.StatusNotFound},
		"CAFETERIA_NOT_ACTIVE":        {ErrCafeteriaNotActive, http.StatusBadRequest},
		"CAFETERIA_NO_DINNER":         {ErrCafeteriaNoDinner, http.StatusBadRequest},
		"CAFETERIA_NO_VEGAN":          {ErrCafeteriaNoVegan, http.StatusBadRequest},
		"RESERVATION_NOT_FOUND":       {ErrReservationNotFound, http.StatusNotFound},
		"NO_RESERVATION":              {ErrNoReservation, http.StatusNotFound},
		"ACTIVE_RESERVATION_EXISTS":   {ErrActiveReservationExists, http.StatusConflict},
		"RESERVATION_ALREADY_USED":    {ErrReservationAlreadyUsed, http.StatusBadRequest},
		"INVALID_STATUS_FOR_CANCEL":   {ErrInvalidStatusForCancel, http.StatusBadRequest},
		"NOT_OWNER":                   {ErrNotOwner, http.StatusForbidden},
		"CANCEL_CUTOFF_PASSED":        {ErrCancelCutoffPassed, http.StatusBadRequest},
		"STUDENT_DEACTIVATED":         {ErrStudentDeactivated, http.StatusForbidden},
		"CAFETERIA_CLOSED":            {ErrCafeteriaClosedOnDate, http.StatusBadRequest},
		"INVALID_DATE_RANGE":          {ErrInvalidDateRange, http.StatusBadRequest},
		"OUTSIDE_RESERVATION_WINDOW":  {ErrOutsideReservationWindow, http.StatusBadRequest},
		"INVALID_MEAL_TIME":           {ErrInvalidMealTime, http.StatusBadRequest},
		"INVALID_MENU_TYPE":           {ErrInvalidMenuType, http.StatusBadRequest},
		"INVALID_QR":                  {ErrInvalidQR, http.StatusBadRequest},
		"INVALID_QR_DATE":             {ErrInvalidQRDate, http.StatusBadRequest},
		"OUTSIDE_MEAL_TIME_WINDOW":    {ErrOutsideMealTimeWindow, http.StatusBadRequest},
		"RESERVATION_CONFLICTS":       {ErrReservationConflicts, http.StatusConflict},
		"VALIDATION_ERRORS":           {ErrValidationErrors, http.StatusBadRequest},
		"PAYMENT_SERVICE_ERROR":       {ErrPaymentServiceError, http.StatusFailedDependency},
		"PAYMENT_FAILED":              {ErrPaymentFailed, http.StatusFailedDependency},
		"REFUND_FAILED":               {ErrRefundFailed, http.StatusFailedDependency},
	}
	for code, c := range cases {
		assert.Equal(t, code, c.err.Code)
		assert.Equal(t, c.want, c.err.HTTPStatus, "status mismatch for %s", code)
		assert.NotEmpty(t, c.err.Message)
	}
}

func TestMealErrors_DistinctCodes(t *testing.T) {
	all := []*sharedErrors.AppError{
		ErrCafeteriaNotFound, ErrCafeteriaNotActive,
		ErrCafeteriaNoDinner, ErrCafeteriaNoVegan,
		ErrReservationNotFound, ErrNoReservation,
		ErrActiveReservationExists, ErrReservationAlreadyUsed,
		ErrInvalidStatusForCancel, ErrNotOwner, ErrCancelCutoffPassed,
		ErrStudentDeactivated,
		ErrCafeteriaClosedOnDate, ErrInvalidDateRange,
		ErrOutsideReservationWindow, ErrInvalidMealTime, ErrInvalidMenuType,
		ErrInvalidQR, ErrInvalidQRDate, ErrOutsideMealTimeWindow,
		ErrReservationConflicts, ErrValidationErrors,
		ErrPaymentServiceError, ErrPaymentFailed, ErrRefundFailed,
	}
	seen := make(map[string]bool, len(all))
	for _, e := range all {
		assert.False(t, seen[e.Code], "duplicate code %q", e.Code)
		seen[e.Code] = true
	}
}
