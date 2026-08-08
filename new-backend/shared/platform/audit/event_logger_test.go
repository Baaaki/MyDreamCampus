package audit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOutbox struct {
	payloads [][]byte
	err      error
}

func (f *fakeOutbox) QueueAuditEvent(_ context.Context, payload []byte) error {
	if f.err != nil {
		return f.err
	}
	f.payloads = append(f.payloads, payload)
	return nil
}

func TestEventAuditLogger_Log_QueuesEntryWithServiceName(t *testing.T) {
	outbox := &fakeOutbox{}
	logger := NewEventAuditLogger(outbox, "grades")

	require.NoError(t, logger.Log(context.Background(), AuditEvent{
		ActorID:      "11111111-1111-1111-1111-111111111111",
		ActorRole:    "admin",
		Action:       "grade.finalized",
		ResourceType: "grade",
		Details:      map[string]any{"course": "BLM101"},
	}))

	require.Len(t, outbox.payloads, 1)
	var got AuditEvent
	require.NoError(t, json.Unmarshal(outbox.payloads[0], &got))

	assert.Equal(t, "grades", got.Service, "the logger stamps the publishing service, callers do not")
	assert.Equal(t, "grade.finalized", got.Action)
	assert.Equal(t, "BLM101", got.Details["course"])
}

func TestEventAuditLogger_Log_OutboxFailurePropagates(t *testing.T) {
	outbox := &fakeOutbox{err: errors.New("db down")}
	logger := NewEventAuditLogger(outbox, "meal")

	err := logger.Log(context.Background(), AuditEvent{Action: "reservation.created"})
	assert.Error(t, err, "callers log a failed audit write as a warning; swallowing it would hide the loss")
}
