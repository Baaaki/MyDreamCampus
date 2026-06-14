package worker

import (
	"testing"

	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/stretchr/testify/assert"
)

// getRoutingKey owns the wire contract between the student outbox row's
// event_type column and the routing key downstream consumers (auth,
// attendance, enrollment, meal) subscribe to. A typo here silently breaks
// every consumer of student.* — pin every case by name.
func TestOutboxWorker_GetRoutingKey(t *testing.T) {
	w := &OutboxWorker{}

	tests := []struct {
		name      string
		eventType string
		want      string
	}{
		{
			name:      "student.created maps to its routing key",
			eventType: events.EventStudentCreated,
			want:      events.RoutingKeyStudentCreated,
		},
		{
			name:      "student.updated maps to its routing key",
			eventType: events.EventStudentUpdated,
			want:      events.RoutingKeyStudentUpdated,
		},
		{
			name:      "student.deactivated maps to its routing key",
			eventType: events.EventStudentDeactivated,
			want:      events.RoutingKeyStudentDeactivated,
		},
		{
			name:      "unknown event type falls back to the event type itself",
			eventType: "student.transferred",
			want:      "student.transferred",
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

// Lock the literal wire format. shared/events houses the constants but the
// publisher's tests pin literals so a downstream consumer search finds both
// ends of the contract in one grep.
func TestOutboxWorker_GetRoutingKey_LiteralWireFormat(t *testing.T) {
	w := &OutboxWorker{}

	assert.Equal(t, "student.created", w.getRoutingKey(events.EventStudentCreated))
	assert.Equal(t, "student.updated", w.getRoutingKey(events.EventStudentUpdated))
	assert.Equal(t, "student.deactivated", w.getRoutingKey(events.EventStudentDeactivated))
}
