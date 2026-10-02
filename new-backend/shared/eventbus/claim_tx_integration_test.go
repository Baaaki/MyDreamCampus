//go:build integration

package eventbus

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const claimTestTimeout = 300 * time.Millisecond

func newClaimTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dir := t.TempDir()
	migration := "-- +goose Up\nCREATE TABLE claim_rows (n int);\n-- +goose Down\nDROP TABLE claim_rows;\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "001_claim_rows.sql"), []byte(migration), 0o600))
	return pgtest.New(t, "claim", dir)
}

func claimRowCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT count(*) FROM claim_rows").Scan(&n))
	return n
}

func insertClaimRow(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "INSERT INTO claim_rows (n) VALUES (1)")
	return err
}

// The timeout belongs to the commit, not to the whole transaction: a broker
// that keeps the batch open for longer than it must not make the final
// commit fail on events it has already confirmed.
func TestInClaimTx_BodyLongerThanTheTimeout_StillCommits(t *testing.T) {
	pool := newClaimTestPool(t)

	err := inClaimTx(context.Background(), pool, claimTestTimeout, func(tx pgx.Tx) error {
		time.Sleep(2 * claimTestTimeout)
		return insertClaimRow(context.Background(), tx)
	})

	require.NoError(t, err)
	assert.Equal(t, 1, claimRowCount(t, pool))
}

// The shutdown signal can land between the broker's confirm and the commit.
func TestInClaimTx_ContextCancelledInsideTheBody_StillCommits(t *testing.T) {
	pool := newClaimTestPool(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := InClaimTx(ctx, pool, func(tx pgx.Tx) error {
		if err := insertClaimRow(context.Background(), tx); err != nil {
			return err
		}
		cancel()
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, claimRowCount(t, pool))
}

func TestInClaimTx_BodyError_RollsBackAndReturnsIt(t *testing.T) {
	pool := newClaimTestPool(t)
	boom := errors.New("boom")

	err := InClaimTx(context.Background(), pool, func(tx pgx.Tx) error {
		if err := insertClaimRow(context.Background(), tx); err != nil {
			return err
		}
		return boom
	})

	require.ErrorIs(t, err, boom)
	assert.Equal(t, 0, claimRowCount(t, pool))
}

func TestInClaimTx_ContextAlreadyCancelled_DoesNotBegin(t *testing.T) {
	pool := newClaimTestPool(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false

	err := InClaimTx(ctx, pool, func(pgx.Tx) error {
		called = true
		return nil
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, called)
}
