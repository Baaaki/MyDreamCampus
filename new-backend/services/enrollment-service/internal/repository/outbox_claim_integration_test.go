//go:build integration

package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/baaaki/mydreamcampus/enrollment/internal/db"
	"github.com/baaaki/mydreamcampus/shared/eventbus"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Two workers polling at the same time must not see the same events: with
// FOR UPDATE SKIP LOCKED each worker locks its batch inside the transaction,
// so the second worker skips the locked rows and gets the next batch.
func TestOutboxStore_ClaimPending_TwoConcurrentClaimersGetDisjointRows(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	pool := pgtest.New(t, "enrollment", "../sql/migrations")
	repo := NewOutboxRepository(pool)
	store := NewOutboxStore(repo)
	ctx := context.Background()

	// Insert 10 pending events
	for i := 0; i < 10; i++ {
		_, err := repo.queries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType:  "test.event",
			RoutingKey: "test.event",
			Payload:    []byte(`{"n":1}`),
		})
		require.NoError(t, err)
	}

	claimer1Hold := make(chan struct{})
	claimer1Started := make(chan struct{})
	var claimer1Events []eventbus.OutboxEvent
	var claimer2Events []eventbus.OutboxEvent

	var wg sync.WaitGroup
	wg.Add(2)

	// Claimer 1 claims 5 rows and holds the transaction open until claimer 2 has run
	go func() {
		defer wg.Done()
		err := store.ClaimPending(ctx, 5, func(cctx context.Context, events []eventbus.OutboxEvent, tx eventbus.ClaimTx) error {
			claimer1Events = events
			close(claimer1Started)
			<-claimer1Hold
			for _, ev := range events {
				_ = tx.MarkProcessed(cctx, ev.ID)
			}
			return nil
		})
		assert.NoError(t, err)
	}()

	// Wait for claimer 1 to acquire its locks
	<-claimer1Started

	// Claimer 2 claims 5 rows while claimer 1's transaction is still active
	go func() {
		defer wg.Done()
		err := store.ClaimPending(ctx, 5, func(cctx context.Context, events []eventbus.OutboxEvent, tx eventbus.ClaimTx) error {
			claimer2Events = events
			for _, ev := range events {
				_ = tx.MarkProcessed(cctx, ev.ID)
			}
			return nil
		})
		assert.NoError(t, err)
		close(claimer1Hold)
	}()

	wg.Wait()

	require.Len(t, claimer1Events, 5)
	require.Len(t, claimer2Events, 5)

	// Verify all claimed IDs are disjoint
	seen := make(map[uuid.UUID]bool)
	for _, ev := range claimer1Events {
		seen[ev.ID] = true
	}
	for _, ev := range claimer2Events {
		assert.False(t, seen[ev.ID], "claimer 2 got an event that claimer 1 already claimed: %s", ev.ID)
	}
	assert.Len(t, seen, 5)
}

func TestOutboxStore_ClaimPending_PartialBatchCommitsConfirmedRowsAndLeavesRestPending(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	pool := pgtest.New(t, "enrollment", "../sql/migrations")
	repo := NewOutboxRepository(pool)
	store := NewOutboxStore(repo)
	ctx := context.Background()

	// Insert 5 pending events
	createdIDs := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		row, err := repo.queries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType:  "test.event",
			RoutingKey: "test.event",
			Payload:    []byte(`{"n":1}`),
		})
		require.NoError(t, err)
		createdIDs[i] = row.ID.Bytes
	}

	// Claim 5, but only process 2 (simulating broker refusal mid-batch)
	err := store.ClaimPending(ctx, 5, func(cctx context.Context, events []eventbus.OutboxEvent, tx eventbus.ClaimTx) error {
		require.Len(t, events, 5)
		require.NoError(t, tx.MarkProcessed(cctx, events[0].ID))
		require.NoError(t, tx.MarkProcessed(cctx, events[1].ID))
		// events[2], [3], [4] are left untouched
		return nil
	})
	require.NoError(t, err)

	// Verify that the remaining 3 events are still pending and retrievable
	remaining, err := repo.GetPendingEvents(ctx, 10)
	require.NoError(t, err)
	require.Len(t, remaining, 3)

	remainingIDs := make([]uuid.UUID, len(remaining))
	for i, r := range remaining {
		remainingIDs[i] = r.ID.Bytes
		assert.Equal(t, int16(0), r.RetryCount.Int16, "retries must not be burned for unhandled events")
	}
	assert.Equal(t, createdIDs[2:], remainingIDs)
}
