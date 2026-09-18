package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeIdempotencyStore struct {
	mu      sync.Mutex
	records map[string]string
	err     error
}

func newFakeIdempotencyStore() *fakeIdempotencyStore {
	return &fakeIdempotencyStore{records: map[string]string{}}
}

func (f *fakeIdempotencyStore) GetIdempotencyRecord(_ context.Context, key string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.records[key], nil
}

func (f *fakeIdempotencyStore) ClaimIdempotencyKey(_ context.Context, key, value string, _ time.Duration) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.records[key]; exists {
		return false, nil
	}
	f.records[key] = value
	return true, nil
}

func (f *fakeIdempotencyStore) SaveIdempotencyRecord(_ context.Context, key, value string, _ time.Duration) error {
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records[key] = value
	return nil
}

func (f *fakeIdempotencyStore) ReleaseIdempotencyKey(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.records, key)
	return nil
}

// newIdempotencyRouter returns a router whose handler counts its own calls.
func newIdempotencyRouter(t *testing.T, store IdempotencyStore) (*gin.Engine, *int) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	SetIdempotencyStore(store, "meal")
	t.Cleanup(func() { globalIdempotency = nil })

	calls := 0
	r := gin.New()
	r.POST("/reservations", Idempotency(), func(c *gin.Context) {
		calls++
		c.JSON(http.StatusCreated, gin.H{"id": "res_1", "call": calls})
	})
	return r, &calls
}

func post(r *gin.Engine, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set(IdempotencyHeader, key)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestIdempotency_SameKeyAndBody_ReplaysStoredResponse(t *testing.T) {
	r, calls := newIdempotencyRouter(t, newFakeIdempotencyStore())
	body := `{"date":"2026-09-01","meal_type":"lunch"}`

	first := post(r, "key-1", body)
	second := post(r, "key-1", body)

	assert.Equal(t, http.StatusCreated, first.Code)
	assert.Equal(t, http.StatusCreated, second.Code)
	assert.JSONEq(t, first.Body.String(), second.Body.String())
	assert.Equal(t, 1, *calls, "the handler must run once, or the retry books a second reservation")
}

func TestIdempotency_SameKeyDifferentBody_Returns422(t *testing.T) {
	r, calls := newIdempotencyRouter(t, newFakeIdempotencyStore())

	post(r, "key-1", `{"date":"2026-09-01"}`)
	reused := post(r, "key-1", `{"date":"2026-09-02"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, reused.Code)
	assert.Contains(t, reused.Body.String(), "IDEMPOTENCY_KEY_REUSED")
	assert.Equal(t, 1, *calls)
}

func TestIdempotency_InFlightKey_Returns409(t *testing.T) {
	store := newFakeIdempotencyStore()
	r, calls := newIdempotencyRouter(t, store)
	body := `{"date":"2026-09-01"}`

	// Simulate the first attempt still running: the claim exists, the
	// response does not.
	claimed, err := store.ClaimIdempotencyKey(t.Context(), "idem:meal:anonymous:key-1",
		`{"body_sha256":"`+sha256Hex([]byte(body))+`","in_flight":true}`, time.Minute)
	require.NoError(t, err)
	require.True(t, claimed)

	w := post(r, "key-1", body)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "IDEMPOTENCY_IN_PROGRESS")
	assert.Equal(t, 0, *calls)
}

func TestIdempotency_NoHeader_PassesThrough(t *testing.T) {
	r, calls := newIdempotencyRouter(t, newFakeIdempotencyStore())

	first := post(r, "", `{"date":"2026-09-01"}`)
	second := post(r, "", `{"date":"2026-09-01"}`)

	assert.Equal(t, http.StatusCreated, first.Code)
	assert.Equal(t, http.StatusCreated, second.Code)
	assert.Equal(t, 2, *calls, "clients that send no key keep the old behaviour")
}

// Redis is not an access control here: losing it must not stop reservations.
func TestIdempotency_StoreUnavailable_FailsOpen(t *testing.T) {
	store := newFakeIdempotencyStore()
	store.err = errors.New("redis down")
	r, calls := newIdempotencyRouter(t, store)

	w := post(r, "key-1", `{"date":"2026-09-01"}`)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, 1, *calls)
}

// A failed attempt must not lock the key for the whole TTL.
func TestIdempotency_HandlerError_ReleasesKey(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	store := newFakeIdempotencyStore()
	SetIdempotencyStore(store, "meal")
	t.Cleanup(func() { globalIdempotency = nil })

	calls := 0
	r := gin.New()
	r.POST("/reservations", Idempotency(), func(c *gin.Context) {
		calls++
		if calls == 1 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "boom"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": "res_1"})
	})

	first := post(r, "key-1", `{"date":"2026-09-01"}`)
	second := post(r, "key-1", `{"date":"2026-09-01"}`)

	assert.Equal(t, http.StatusInternalServerError, first.Code)
	assert.Equal(t, http.StatusCreated, second.Code)
	assert.Equal(t, 2, calls)
}

func TestIdempotency_GetRequest_Ignored(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	store := newFakeIdempotencyStore()
	SetIdempotencyStore(store, "meal")
	t.Cleanup(func() { globalIdempotency = nil })

	r := gin.New()
	r.GET("/reservations", Idempotency(), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/reservations", nil)
	req.Header.Set(IdempotencyHeader, "key-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, store.records, "reads are idempotent already; no Redis round-trip for them")
}

// ctxAwareStore fails writes whose context is already cancelled, the way a
// real Redis client does.
type ctxAwareStore struct {
	*fakeIdempotencyStore
	claimTTL time.Duration
}

func (s *ctxAwareStore) ClaimIdempotencyKey(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	s.claimTTL = ttl
	return s.fakeIdempotencyStore.ClaimIdempotencyKey(ctx, key, value, ttl)
}

func (s *ctxAwareStore) SaveIdempotencyRecord(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.fakeIdempotencyStore.SaveIdempotencyRecord(ctx, key, value, ttl)
}

func TestIdempotency_ClientHangsUp_ResponseStillStored(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	store := &ctxAwareStore{fakeIdempotencyStore: newFakeIdempotencyStore()}
	SetIdempotencyStore(store, "meal")
	t.Cleanup(func() { globalIdempotency = nil })

	ctx, hangUp := context.WithCancel(context.Background())
	r := gin.New()
	r.POST("/reservations", Idempotency(), func(c *gin.Context) {
		hangUp() // the client disconnects while the work completes
		c.JSON(http.StatusCreated, gin.H{"id": "res_1"})
	})

	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(`{"meal":"lunch"}`)).WithContext(ctx)
	req.Header.Set(IdempotencyHeader, "key-hangup")
	r.ServeHTTP(httptest.NewRecorder(), req)

	var stored string
	for _, v := range store.records {
		stored = v
	}
	assert.Contains(t, stored, `"status_code":201`, "the retry must replay the result, not see a stuck in-flight claim")
	assert.NotContains(t, stored, `"in_flight":true`)
}

func TestIdempotency_InFlightClaim_ExpiresQuickly(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	store := &ctxAwareStore{fakeIdempotencyStore: newFakeIdempotencyStore()}
	SetIdempotencyStore(store, "meal")
	t.Cleanup(func() { globalIdempotency = nil })

	r := gin.New()
	r.POST("/reservations", Idempotency(), func(c *gin.Context) { c.Status(http.StatusCreated) })
	req := httptest.NewRequest(http.MethodPost, "/reservations", strings.NewReader(`{}`))
	req.Header.Set(IdempotencyHeader, "key-ttl")
	r.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, inFlightTTL, store.claimTTL,
		"a claim orphaned by a crash must not block the key for the full replay TTL")
}
