package repository

import (
	"context"
	"sync"
	"time"

	"github.com/baaaki/mydreamcampus/meal-service/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// ClosedDaysCache wraps ClosedDaysRepository with an in-memory TTL cache for
// IsDateClosed. Closed days rarely change (admin edits), so the reservation
// hot path can answer from memory most of the time. Writes invalidate the
// entire cache.
type ClosedDaysCache struct {
	inner *ClosedDaysRepository
	ttl   time.Duration

	mu      sync.RWMutex
	entries map[string]closedDayEntry
}

type closedDayEntry struct {
	closed    bool
	expiresAt time.Time
}

func NewClosedDaysCache(inner *ClosedDaysRepository, ttl time.Duration) *ClosedDaysCache {
	return &ClosedDaysCache{
		inner:   inner,
		ttl:     ttl,
		entries: make(map[string]closedDayEntry),
	}
}

func (c *ClosedDaysCache) IsDateClosed(ctx context.Context, date pgtype.Date) (bool, error) {
	key := date.Time.Format("2006-01-02")

	c.mu.RLock()
	if entry, ok := c.entries[key]; ok && time.Now().Before(entry.expiresAt) {
		c.mu.RUnlock()
		return entry.closed, nil
	}
	c.mu.RUnlock()

	closed, err := c.inner.IsDateClosed(ctx, date)
	if err != nil {
		return false, err
	}

	c.mu.Lock()
	c.entries[key] = closedDayEntry{closed: closed, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()

	return closed, nil
}

// Invalidate drops every cached entry. Call after any write to closed_days.
func (c *ClosedDaysCache) Invalidate() {
	c.mu.Lock()
	c.entries = make(map[string]closedDayEntry)
	c.mu.Unlock()
}

// Pass-through methods keep the cache as a drop-in replacement.

func (c *ClosedDaysCache) CreateClosedDay(ctx context.Context, params db.CreateClosedDayParams) (db.ClosedDay, error) {
	result, err := c.inner.CreateClosedDay(ctx, params)
	if err == nil {
		c.Invalidate()
	}
	return result, err
}

func (c *ClosedDaysCache) DeleteClosedDay(ctx context.Context, id pgtype.UUID) error {
	if err := c.inner.DeleteClosedDay(ctx, id); err != nil {
		return err
	}
	c.Invalidate()
	return nil
}

func (c *ClosedDaysCache) ListClosedDays(ctx context.Context, params db.ListClosedDaysParams) ([]db.ClosedDay, error) {
	return c.inner.ListClosedDays(ctx, params)
}

func (c *ClosedDaysCache) GetClosedDaysByDateRange(ctx context.Context, params db.GetClosedDaysByDateRangeParams) ([]db.ClosedDay, error) {
	return c.inner.GetClosedDaysByDateRange(ctx, params)
}

func (c *ClosedDaysCache) DeleteClosedDaysBySemester(ctx context.Context, semester string) error {
	if err := c.inner.DeleteClosedDaysBySemester(ctx, semester); err != nil {
		return err
	}
	c.Invalidate()
	return nil
}
