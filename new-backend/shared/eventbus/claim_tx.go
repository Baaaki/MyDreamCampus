package eventbus

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// TxBeginner is the part of *pgxpool.Pool the claim transaction needs.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// InClaimTx runs fn inside a transaction on db and commits it when fn
// returns nil, rolling it back on an error.
//
// The relay publishes to the broker while this transaction is open, then
// records what the broker confirmed. Two things follow for the commit: it
// must not die with ctx, because a shutdown signal arriving right after a
// confirm would roll back rows the broker already holds and publish them a
// second time on the next start; and its timeout must be counted from the
// commit itself, not from BEGIN, because a slow broker can keep the batch
// open for longer than the timeout and the commit would then fail on every
// event it had just sent. Begin, commit and rollback therefore each get a
// timeout of their own on a context detached from ctx's cancellation.
func InClaimTx(ctx context.Context, db TxBeginner, fn func(tx pgx.Tx) error) error {
	return inClaimTx(ctx, db, SettleTimeout, fn)
}

func inClaimTx(ctx context.Context, db TxBeginner, timeout time.Duration, fn func(tx pgx.Tx) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	detached := context.WithoutCancel(ctx)

	beginCtx, cancelBegin := context.WithTimeout(detached, timeout)
	tx, err := db.Begin(beginCtx)
	cancelBegin()
	if err != nil {
		return err
	}

	// After a successful commit the rollback finds the transaction closed and
	// does nothing, so it can run unconditionally, panics in fn included.
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(detached, timeout)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()

	if err := fn(tx); err != nil {
		return err
	}

	commitCtx, cancelCommit := context.WithTimeout(detached, timeout)
	defer cancelCommit()
	return tx.Commit(commitCtx)
}
