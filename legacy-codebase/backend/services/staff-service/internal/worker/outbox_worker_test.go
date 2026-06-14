package worker

import (
	"testing"

	"github.com/baaaki/mydreamcampus/shared/events"
	"github.com/stretchr/testify/assert"
)

// getRoutingKey is the only pure piece of OutboxWorker — it owns the wire
// contract between the outbox event_type column and the RabbitMQ routing key
// every downstream consumer subscribes to. A typo here silently breaks every
// service that listens on staff.* (auth, student, course-catalog, etc.) so
// the mapping is pinned by name, not by re-deriving from the constant.
func TestOutboxWorker_GetRoutingKey(t *testing.T) {
	w := &OutboxWorker{}

	tests := []struct {
		name      string
		eventType string
		want      string
	}{
		{
			name:      "staff.created maps to its routing key constant",
			eventType: events.EventStaffCreated,
			want:      events.RoutingKeyStaffCreated,
		},
		{
			name:      "staff.updated maps to its routing key constant",
			eventType: events.EventStaffUpdated,
			want:      events.RoutingKeyStaffUpdated,
		},
		{
			name:      "staff.deactivated maps to its routing key constant",
			eventType: events.EventStaffDeactivated,
			want:      events.RoutingKeyStaffDeactivated,
		},
		{
			name:      "unknown event type falls back to the event type itself",
			eventType: "staff.something.new",
			want:      "staff.something.new",
		},
		{
			name:      "empty event type passes through (no panic, no default routing)",
			eventType: "",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.getRoutingKey(tt.eventType)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Lock the literal routing key strings — these go on the wire to RabbitMQ
// and any rename would silently re-route messages. The constants live in
// shared/events but the assertion belongs in the publisher-side test so a
// downstream consumer can search by literal and find both ends.
func TestOutboxWorker_GetRoutingKey_LiteralWireFormat(t *testing.T) {
	w := &OutboxWorker{}

	assert.Equal(t, "staff.created", w.getRoutingKey(events.EventStaffCreated))
	assert.Equal(t, "staff.updated", w.getRoutingKey(events.EventStaffUpdated))
	assert.Equal(t, "staff.deactivated", w.getRoutingKey(events.EventStaffDeactivated))
}
