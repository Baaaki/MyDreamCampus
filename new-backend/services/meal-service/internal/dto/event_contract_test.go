package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wire contracts for events meal-service produces and consumes.
// Producers / consumers on the other side of each event:
//   student.* (in)         ← student-service worker
//   payment.completed (in) ← payment-service mock
//   payment.failed    (in) ← payment-service mock
//   meal.reservation.* (out) → currently no consumer; pinning anyway so a
//                              future addition can rely on the names.
//
// These tests exist because the project has already had a silent DTO mismatch
// (course_id → semester_course_id rename caught by the catalog/attendance
// contract pair). The same risk lives here for student.* events; pin both
// the wrapped shape and the data.id field name.

const studentCreatedFixture = `{
  "event_id": "00000000-0000-0000-0000-0000000000aa",
  "event_type": "student.created",
  "timestamp": "2026-04-27T09:00:00Z",
  "data": {
    "id": "11111111-1111-1111-1111-111111111111",
    "student_number": "20210001",
    "first_name": "Ahmet",
    "last_name": "Yilmaz"
  }
}`

func TestStudentCreatedEvent_DecodesWrapped(t *testing.T) {
	var got StudentCreatedEvent
	require.NoError(t, json.Unmarshal([]byte(studentCreatedFixture), &got))

	assert.Equal(t, "student.created", got.EventType)
	// student-service emits the row's primary key as data.id (NOT data.student_id).
	// meal-service mirrors that tag — pin so a producer-side rename breaks here.
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", got.Data.ID,
		"json tag is 'id', not 'student_id' — historical contract from student-service")
	assert.Equal(t, "20210001", got.Data.StudentNumber)
	assert.Equal(t, "Ahmet", got.Data.FirstName)
}

func TestStudentCreatedEvent_FlatPayloadFails(t *testing.T) {
	// Regression class: if a future producer flattens this event, every
	// data-side field zeroes. Document the failure mode so the fix is
	// obvious if anyone bisects to this test.
	flat := `{
      "event_id": "00000000-0000-0000-0000-0000000000aa",
      "event_type": "student.created",
      "id": "11111111-1111-1111-1111-111111111111",
      "student_number": "20210001"
    }`

	var got StudentCreatedEvent
	require.NoError(t, json.Unmarshal([]byte(flat), &got))

	// Wrapper still decodes, but Data is zero — proving the contract is
	// strictly wrapped.
	assert.Equal(t, "student.created", got.EventType)
	assert.Empty(t, got.Data.ID)
	assert.Empty(t, got.Data.StudentNumber)
}

func TestStudentUpdatedEvent_DecodesWrapped(t *testing.T) {
	body := `{
      "event_id": "00000000-0000-0000-0000-0000000000bb",
      "event_type": "student.updated",
      "timestamp": "2026-04-27T09:00:00Z",
      "data": {
        "id": "11111111-1111-1111-1111-111111111111",
        "student_number": "20210001",
        "first_name": "Mehmet",
        "last_name": "Yilmaz"
      }
    }`
	var got StudentUpdatedEvent
	require.NoError(t, json.Unmarshal([]byte(body), &got))
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", got.Data.ID)
	assert.Equal(t, "Mehmet", got.Data.FirstName)
}

func TestStudentDeactivatedEvent_OnlyHasID(t *testing.T) {
	// Deactivation only carries the row PK — the consumer flips the cached
	// student row to inactive. If the producer ever ships extra data here it's
	// fine, but if the id field name changes the consumer silently no-ops.
	body := `{
      "event_id": "deact-1",
      "event_type": "student.deactivated",
      "timestamp": "2026-04-27T09:00:00Z",
      "data": {"id": "11111111-1111-1111-1111-111111111111"}
    }`
	var got StudentDeactivatedEvent
	require.NoError(t, json.Unmarshal([]byte(body), &got))
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", got.Data.ID)
}

// =============================================================================
// payment.* — consumed from payment-service (mock)
// =============================================================================

func TestPaymentCompletedEvent_DecodesWrapped(t *testing.T) {
	body := `{
      "event_id": "00000000-0000-0000-0000-0000000000cc",
      "event_type": "payment.completed",
      "timestamp": "2026-04-27T09:00:00Z",
      "data": {
        "payment_id": "pay_123",
        "reference_id": "res_22222222-2222-2222-2222-222222222222",
        "amount": 45.5,
        "currency": "TRY"
      }
    }`
	var got PaymentCompletedEvent
	require.NoError(t, json.Unmarshal([]byte(body), &got))

	// reference_id contains the prefix "res_" or "bat_" — meal-service routes
	// based on that prefix in the consumer. Pin the format documentation here.
	assert.Equal(t, "res_22222222-2222-2222-2222-222222222222", got.Data.ReferenceID,
		"reference_id shape is '<prefix>_<uuid>' where prefix is 'res' or 'bat'")
	assert.Equal(t, 45.5, got.Data.Amount)
	assert.Equal(t, "TRY", got.Data.Currency)
}

func TestPaymentFailedEvent_DecodesWrapped(t *testing.T) {
	body := `{
      "event_id": "00000000-0000-0000-0000-0000000000dd",
      "event_type": "payment.failed",
      "timestamp": "2026-04-27T09:00:00Z",
      "data": {
        "payment_id": "pay_456",
        "reference_id": "bat_22222222-2222-2222-2222-222222222222",
        "reason": "insufficient_funds"
      }
    }`
	var got PaymentFailedEvent
	require.NoError(t, json.Unmarshal([]byte(body), &got))

	assert.Equal(t, "bat_22222222-2222-2222-2222-222222222222", got.Data.ReferenceID)
	assert.Equal(t, "insufficient_funds", got.Data.Reason)
}

// =============================================================================
// OUTBOUND — published by meal-service via outbox
// =============================================================================

func TestMealReservationCreatedEvent_PublishedShape(t *testing.T) {
	evt := MealReservationCreatedEvent{
		EventType: "meal.reservation.created",
		EventID:   "evt-1",
		Data: MealReservationCreatedEventData{
			ReservationID: "res-1",
			StudentID:     "stu-1",
			StudentNumber: "20210001",
			Date:          "2026-05-01",
			MealTime:      "lunch",
			MenuType:      "regular",
			CafeteriaID:   "caf-1",
			CafeteriaName: "Main Hall",
			Amount:        35.0,
			Currency:      "TRY",
		},
	}

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Contains(t, decoded, "data")

	dataMap := decoded["data"].(map[string]any)
	for _, k := range []string{
		"reservation_id", "student_id", "student_number", "date",
		"meal_time", "menu_type", "cafeteria_id", "cafeteria_name",
		"amount", "currency",
	} {
		assert.Contains(t, dataMap, k, "data must include %q", k)
	}
	assert.Equal(t, "lunch", dataMap["meal_time"])
	assert.Equal(t, "TRY", dataMap["currency"])
}

func TestMealReservationCancelledEvent_PublishedShape(t *testing.T) {
	evt := MealReservationCancelledEvent{
		EventType: "meal.reservation.cancelled",
		Data: MealReservationCancelledEventData{
			ReservationID: "res-1",
			StudentID:     "stu-1",
			StudentNumber: "20210001",
			Date:          "2026-05-01",
			MealTime:      "lunch",
			RefundAmount:  35.0,
			Currency:      "TRY",
		},
	}

	raw, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	dataMap := decoded["data"].(map[string]any)
	for _, k := range []string{
		"reservation_id", "student_id", "student_number", "date",
		"meal_time", "refund_amount", "currency",
	} {
		assert.Contains(t, dataMap, k)
	}
	assert.Equal(t, float64(35.0), dataMap["refund_amount"])
}
