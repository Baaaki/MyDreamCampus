package eventbus

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type outboxStatus string

const (
	statusPending   outboxStatus = "pending"
	statusProcessed outboxStatus = "processed"
	statusFailed    outboxStatus = "failed"
)

type fakeRow struct {
	event  OutboxEvent
	status outboxStatus
}

// fakeStore is an in-memory outbox that keeps what a real table keeps: a
// status and a retry counter per row.
type fakeStore struct {
	mu   sync.Mutex
	rows []*fakeRow
}

var _ OutboxStore = (*fakeStore)(nil)

func newFakeStore(payloads ...string) *fakeStore {
	s := &fakeStore{}
	for _, p := range payloads {
		s.rows = append(s.rows, &fakeRow{
			event:  OutboxEvent{ID: uuid.New(), EventType: "thing.created", RoutingKey: "thing.created", Payload: []byte(p), MaxRetries: 5},
			status: statusPending,
		})
	}
	return s
}

func (s *fakeStore) GetPending(ctx context.Context, limit int32) ([]OutboxEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []OutboxEvent
	for _, r := range s.rows {
		if r.status == statusPending && int32(len(out)) < limit {
			out = append(out, r.event)
		}
	}
	return out, nil
}

func (s *fakeStore) GetFailed(ctx context.Context, limit int32) ([]OutboxEvent, error) {
	return nil, nil
}

func (s *fakeStore) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.set(id, statusProcessed, false)
}

func (s *fakeStore) MarkFailed(ctx context.Context, id uuid.UUID, msg string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.set(id, statusFailed, true)
}

func (s *fakeStore) Reset(ctx context.Context, id uuid.UUID) error { return nil }

func (s *fakeStore) set(id uuid.UUID, st outboxStatus, bumpRetry bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.rows {
		if r.event.ID == id {
			r.status = st
			if bumpRetry {
				r.event.RetryCount++
			}
			return nil
		}
	}
	return errors.New("no such row")
}

func (s *fakeStore) statuses() []outboxStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]outboxStatus, len(s.rows))
	for i, r := range s.rows {
		out[i] = r.status
	}
	return out
}

func (s *fakeStore) retries() []int16 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]int16, len(s.rows))
	for i, r := range s.rows {
		out[i] = r.event.RetryCount
	}
	return out
}

// fakeBroker confirms every publish until failFrom publishes have been
// accepted, then refuses the rest the way an unreachable broker does.
type fakeBroker struct {
	mu       sync.Mutex
	failFrom int // 0-based publish number from which publishes fail; -1 never
	onCall   func(n int)
	calls    int
	keys     []string
}

func (b *fakeBroker) Publish(ctx context.Context, exchange, routingKey string, payload any) error {
	b.mu.Lock()
	n := b.calls
	b.calls++
	b.mu.Unlock()
	if b.onCall != nil {
		b.onCall(n)
	}
	if b.failFrom >= 0 && n >= b.failFrom {
		return errors.New("rabbitmq: publish not confirmed by the broker")
	}
	b.mu.Lock()
	b.keys = append(b.keys, routingKey)
	b.mu.Unlock()
	return nil
}

func newTestWorker(t *testing.T, store OutboxStore, broker Publisher, batch int32) *OutboxWorker {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	return NewOutboxWorker("test", "test.events", store, broker, time.Hour, batch)
}

// A refused publish is the broker's fault, not the event's. The old relay
// marked the row failed and burned one of its retries on it, so about 25
// seconds of outage parked every event in the batch for good.
func TestProcessEvents_BrokerDown_LeavesEventsPendingWithRetriesUntouched(t *testing.T) {
	store := newFakeStore(`{"n":1}`, `{"n":2}`, `{"n":3}`)
	broker := &fakeBroker{failFrom: 0}

	newTestWorker(t, store, broker, 10).processEvents(context.Background())

	assert.Equal(t, []outboxStatus{statusPending, statusPending, statusPending}, store.statuses())
	assert.Equal(t, []int16{0, 0, 0}, store.retries(), "an outage must not eat the event's retries")
}

// Every further publish would sit out its own confirm timeout, so a batch is
// abandoned at the first refusal instead of hammering a broker that is down.
func TestProcessEvents_BrokerDown_StopsTheBatchAtTheFirstRefusal(t *testing.T) {
	store := newFakeStore(`{"n":1}`, `{"n":2}`, `{"n":3}`)
	broker := &fakeBroker{failFrom: 0}

	newTestWorker(t, store, broker, 10).processEvents(context.Background())

	assert.Equal(t, 1, broker.calls)
}

func TestProcessEvents_BrokerFailsMidBatch_KeepsWhatWasConfirmed(t *testing.T) {
	store := newFakeStore(`{"n":1}`, `{"n":2}`, `{"n":3}`)
	broker := &fakeBroker{failFrom: 2}

	newTestWorker(t, store, broker, 10).processEvents(context.Background())

	assert.Equal(t, []outboxStatus{statusProcessed, statusProcessed, statusPending}, store.statuses())
	assert.Equal(t, []int16{0, 0, 0}, store.retries())
}

// A payload that is not JSON is the event's own fault: no broker will ever
// take it, so it is parked and the rest of the batch goes on.
func TestProcessEvents_MalformedPayload_IsParkedAndTheBatchContinues(t *testing.T) {
	store := newFakeStore(`not json`, `{"n":2}`)
	broker := &fakeBroker{failFrom: -1}

	newTestWorker(t, store, broker, 10).processEvents(context.Background())

	assert.Equal(t, []outboxStatus{statusFailed, statusProcessed}, store.statuses())
	assert.Equal(t, 1, broker.calls)
}
