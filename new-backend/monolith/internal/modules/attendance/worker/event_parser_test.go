package worker

import (
	"encoding/json"
	"testing"

	"github.com/baaaki/mydreamcampus/monolith/internal/modules/attendance/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the consumer side of the wire contract for student.created
// (and by extension every event using BaseEvent envelope). The matching
// producer-side test lives in
//   backend/services/student-service/internal/service/event_payloads_test.go.

func TestUnwrapEventData_StudentCreated_HappyPath(t *testing.T) {
	studentID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	body := mustMarshal(t, map[string]any{
		"event_id":   uuid.New().String(),
		"event_type": "student.created",
		"timestamp":  "2026-04-28T10:00:00Z",
		"data": map[string]any{
			"id":             studentID.String(),
			"student_number": "20210001",
			"first_name":     "Ahmet",
			"last_name":      "Yilmaz",
			"email":          "a@x.x",
			"department":     "CS",
		},
	})

	got, err := unwrapEventData[dto.StudentCreatedEventData](body)
	require.NoError(t, err)
	assert.Equal(t, studentID, got.StudentID)
	assert.Equal(t, "20210001", got.StudentNumber)
	assert.Equal(t, "Ahmet", got.FirstName)
	assert.Equal(t, "CS", got.Department)
}

func TestUnwrapEventData_BadEnvelope(t *testing.T) {
	_, err := unwrapEventData[dto.StudentCreatedEventData]([]byte("not json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base envelope",
		"failure mode must clearly point at the envelope so on-call can tell envelope drift from data drift")
}

func TestUnwrapEventData_DataFieldMissing(t *testing.T) {
	// A producer that drops the `data` key entirely. base.Data ends up as
	// nil, which round-trips to "null"; unmarshalling "null" into a struct
	// is a no-op, leaving zero values. This is the documented behavior:
	// no panic, but the caller gets a zero struct it must validate before
	// using. Test pins the contract.
	body := mustMarshal(t, map[string]any{
		"event_id":   uuid.New().String(),
		"event_type": "student.created",
	})

	got, err := unwrapEventData[dto.StudentCreatedEventData](body)
	require.NoError(t, err, "missing data must NOT cause an error — handlers downstream rely on zero-value detection")
	assert.Equal(t, dto.StudentCreatedEventData{}, got, "absent data must yield the zero value")
}

func TestUnwrapEventData_TypeMismatch(t *testing.T) {
	// student_number is a string in the contract — sending an int there is
	// the canonical contract drift scenario. Locking in that the parser
	// errors loudly rather than silently coercing.
	body := mustMarshal(t, map[string]any{
		"event_id":   uuid.New().String(),
		"event_type": "student.created",
		"data": map[string]any{
			"id":             uuid.New().String(),
			"student_number": 12345, // wrong type
		},
	})

	_, err := unwrapEventData[dto.StudentCreatedEventData](body)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "typed data did not match contract")
}

func TestUnwrapEventData_RoundTripWithStudentUpdated(t *testing.T) {
	// Same helper, different typed payload — proves the generic dispatch
	// works for every consumer that uses the BaseEvent envelope, not just
	// student.created.
	studentID := uuid.New()
	body := mustMarshal(t, map[string]any{
		"event_id":   uuid.New().String(),
		"event_type": "student.updated",
		"data": map[string]any{
			"id":             studentID.String(),
			"student_number": "20210001",
			"first_name":     "Ahmet",
			"last_name":      "Yilmaz",
			"email":          "a@x.x",
			"department":     "CS",
		},
	})

	got, err := unwrapEventData[dto.StudentUpdatedEventData](body)
	require.NoError(t, err)
	assert.Equal(t, studentID, got.StudentID)
	assert.Equal(t, "20210001", got.StudentNumber)
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
