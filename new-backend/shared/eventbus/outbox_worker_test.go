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
	// markProcessedErr, when set, makes every "mark sent" fail the way a
	// database that went away mid-batch does.
	markProcessedErr error
	// markFailedErr does the same for "park this event".
	markFailedErr error
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

type fakeClaimTx struct {
	store *fakeStore
}

func (tx *fakeClaimTx) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), SettleTimeout)
	defer cancel()
	return tx.store.MarkProcessed(settleCtx, id)
}

func (tx *fakeClaimTx) MarkFailed(ctx context.Context, id uuid.UUID, msg string) error {
	settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), SettleTimeout)
	defer cancel()
	return tx.store.MarkFailed(settleCtx, id, msg)
}

func (s *fakeStore) ClaimPending(ctx context.Context, limit int32, settle func(ctx context.Context, events []OutboxEvent, tx ClaimTx) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	var out []OutboxEvent
	for _, r := range s.rows {
		if r.status == statusPending && int32(len(out)) < limit {
			out = append(out, r.event)
		}
	}
	s.mu.Unlock()

	if len(out) == 0 {
		return settle(ctx, nil, nil)
	}

	claimTx := &fakeClaimTx{store: s}
	return settle(ctx, out, claimTx)
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
	if s.markProcessedErr != nil {
		return s.markProcessedErr
	}
	return s.set(id, statusProcessed, false)
}

func (s *fakeStore) MarkFailed(ctx context.Context, id uuid.UUID, msg string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.markFailedErr != nil {
		return s.markFailedErr
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
	if err := ctx.Err(); err != nil {
		return err
	}
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

// The broker's confirm and the shutdown signal can arrive back to back. The
// event is then already out, so recording it must not depend on the context
// that was just cancelled: a row left pending is published a second time on
// the next start.
func TestProcessEvents_ShutdownRightAfterConfirm_StillRecordsTheEventAsSent(t *testing.T) {
	store := newFakeStore(`{"n":1}`, `{"n":2}`, `{"n":3}`)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	broker := &fakeBroker{failFrom: -1, onCall: func(n int) {
		if n == 0 {
			cancel()
		}
	}}

	newTestWorker(t, store, broker, 10).processEvents(ctx)

	assert.Equal(t, []outboxStatus{statusProcessed, statusPending, statusPending}, store.statuses())
	assert.Equal(t, 1, broker.calls, "nothing more goes out once the context is cancelled")
}

func payloads(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = `{"n":1}`
	}
	return out
}

// One batch per tick capped each service at batch/interval events a second
// (10 per 5 s), so a burst of enrollments took minutes to reach its
// consumers. A tick now runs until a batch comes back short.
func TestProcessEvents_BacklogLargerThanABatch_IsDrainedInOneTick(t *testing.T) {
	store := newFakeStore(payloads(25)...)
	broker := &fakeBroker{failFrom: -1}

	newTestWorker(t, store, broker, 10).processEvents(context.Background())

	assert.Equal(t, 25, broker.calls)
	for _, st := range store.statuses() {
		assert.Equal(t, statusProcessed, st)
	}
}

// Inside the claim transaction a failed write aborts the transaction, so
// every later write fails too and the whole batch rolls back. Publishing the
// rest of it would only send events the next tick sends again, and a row whose
// "mark sent" keeps failing is still pending, so reading the next batch would
// hand the same rows back and the drain would never end.
func TestProcessEvents_MarkSentFails_StopsTheBatchAtTheFirstFailedWrite(t *testing.T) {
	store := newFakeStore(payloads(25)...)
	store.markProcessedErr = errors.New("connection reset")
	broker := &fakeBroker{failFrom: -1}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	newTestWorker(t, store, broker, 10).processEvents(ctx)

	assert.Equal(t, 1, broker.calls, "the event whose write failed is the last one sent")
}

// Same for parking a malformed payload: the transaction is poisoned, so the
// valid event behind it must not be published only to roll back.
func TestProcessEvents_ParkingFails_StopsTheBatchAtTheFirstFailedWrite(t *testing.T) {
	store := newFakeStore(`not json`, `{"n":2}`)
	store.markFailedErr = errors.New("connection reset")
	broker := &fakeBroker{failFrom: -1}

	newTestWorker(t, store, broker, 10).processEvents(context.Background())

	assert.Equal(t, 0, broker.calls)
}
