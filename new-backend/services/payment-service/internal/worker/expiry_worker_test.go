package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type countingExpirer struct{ calls atomic.Int32 }

func (e *countingExpirer) ExpireOverdue(context.Context) (int64, error) {
	e.calls.Add(1)
	return 0, nil
}

// A restart must not leave overdue payments open for a whole interval.
func TestExpiryWorker_Start_RunsImmediatelyAndStopsOnCancel(t *testing.T) {
	expirer := &countingExpirer{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		NewExpiryWorker(expirer, time.Hour, zap.NewNop()).Start(ctx)
		close(done)
	}()

	assert.Eventually(t, func() bool { return expirer.calls.Load() == 1 }, time.Second, 5*time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after cancel")
	}
}
