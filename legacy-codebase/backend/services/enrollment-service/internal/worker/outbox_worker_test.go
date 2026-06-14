package worker

import (
	"testing"

	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/stretchr/testify/assert"
)

// getRoutingKey owns the wire contract between the enrollment outbox row's
// event_type column and the routing key downstream consumers (grades,
// student, attendance) subscribe to. A typo silently re-routes — pin every
// case by name.
func TestOutboxWorker_GetRoutingKey(t *testing.T) {
	w := &OutboxWorker{}

	tests := []struct {
		name      string
		eventType string
		want      string
	}{
		{
			name:      "enrollment.program.submitted maps to its routing key",
			eventType: events.EventEnrollmentProgramSubmitted,
			want:      events.RoutingKeyEnrollmentProgramSubmitted,
		},
		{
			name:      "enrollment.program.approved maps to its routing key",
			eventType: events.EventEnrollmentProgramApproved,
			want:      events.RoutingKeyEnrollmentProgramApproved,
		},
		{
			name:      "enrollment.program.rejected maps to its routing key",
			eventType: events.EventEnrollmentProgramRejected,
			want:      events.RoutingKeyEnrollmentProgramRejected,
		},
		{
			name:      "enrollment.program.cancelled maps to its routing key",
			eventType: events.EventEnrollmentProgramCancelled,
			want:      events.RoutingKeyEnrollmentProgramCancelled,
		},
		{
			name:      "unknown event type falls back to the event type itself",
			eventType: "enrollment.program.future",
			want:      "enrollment.program.future",
		},
		{
			name:      "empty event type passes through (no panic, no default)",
			eventType: "",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, w.getRoutingKey(tt.eventType))
		})
	}
}

// Lock literal wire format. The constants live in shared/events but the
// publisher's tests must also pin literals so a downstream consumer search
// finds both ends of the contract.
func TestOutboxWorker_GetRoutingKey_LiteralWireFormat(t *testing.T) {
	w := &OutboxWorker{}

	assert.Equal(t, "enrollment.program.submitted", w.getRoutingKey(events.EventEnrollmentProgramSubmitted))
	assert.Equal(t, "enrollment.program.approved", w.getRoutingKey(events.EventEnrollmentProgramApproved))
	assert.Equal(t, "enrollment.program.rejected", w.getRoutingKey(events.EventEnrollmentProgramRejected))
	assert.Equal(t, "enrollment.program.cancelled", w.getRoutingKey(events.EventEnrollmentProgramCancelled))
}
