package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/audit"
	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	"github.com/baaaki/mydreamcampus/shared/platform/clocksync"
	"github.com/baaaki/mydreamcampus/shared/platform/dto"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClockBackend struct {
	value    string
	storeErr error
}

func (f *fakeClockBackend) GetClockState(context.Context, string) (string, error) {
	return f.value, nil
}

func (f *fakeClockBackend) StoreClockState(_ context.Context, _, _, value string, _ time.Duration) error {
	if f.storeErr != nil {
		return f.storeErr
	}
	f.value = value
	return nil
}

func (f *fakeClockBackend) DeleteClockState(context.Context, string, string) error {
	if f.storeErr != nil {
		return f.storeErr
	}
	f.value = ""
	return nil
}

func (f *fakeClockBackend) WatchClockChanges(context.Context, string) <-chan struct{} {
	return nil
}

type recordingAuditLogger struct {
	mu     sync.Mutex
	events []audit.AuditEvent
}

func (r *recordingAuditLogger) Log(_ context.Context, e audit.AuditEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	return nil
}

func setupTimeTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	require.NoError(t, logger.Init("test"))
	clock.Reset()
	t.Cleanup(clock.Reset)
}

// newTimeControlRouter stands in for JWTAuth by setting the claims it would.
func newTimeControlRouter(h *TimeControlHandler) *gin.Engine {
	r := gin.New()
	admin := r.Group("/admin", func(c *gin.Context) {
		c.Set("user_id", "admin-1")
		c.Set("role", "admin")
	})
	h.RegisterRoutes(admin)
	admin.GET("/time/status", TimeStatus("catalog"))
	return r
}

func doJSON(t *testing.T, r http.Handler, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out), w.Body.String())
	return w, out
}

func simulateBody(at time.Time) string {
	return `{"time":"` + at.Format(time.RFC3339) + `"}`
}

func TestTimeStatus_NoSimulation_ReportsRealClock(t *testing.T) {
	setupTimeTest(t)
	r := gin.New()
	r.GET("/admin/time/status", TimeStatus("grades"))

	req := httptest.NewRequest("GET", "/admin/time/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got dto.TimeStatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "grades", got.Service)
	assert.False(t, got.Active)
	assert.Equal(t, int64(0), got.OffsetSeconds)
	assert.True(t, got.CurrentTime.Equal(got.RealTime))
	assert.WithinDuration(t, time.Now(), got.CurrentTime, time.Second)
	assert.Nil(t, got.Until)
}

func TestTimeStatus_Simulation_ReportsOffsetFromOneSnapshot(t *testing.T) {
	setupTimeTest(t)
	until := time.Now().Add(time.Hour).Truncate(time.Second)
	clock.SetOffset(-90*time.Minute, until)
	r := gin.New()
	r.GET("/admin/time/status", TimeStatus("meal"))

	req := httptest.NewRequest("GET", "/admin/time/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var got dto.TimeStatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.True(t, got.Active)
	assert.Equal(t, int64(-5400), got.OffsetSeconds)
	assert.Equal(t, -90*time.Minute, got.CurrentTime.Sub(got.RealTime))
	require.NotNil(t, got.Until)
	assert.True(t, until.Equal(*got.Until))
}

func TestTimeControl_Simulate_SharesOffsetAndAudits(t *testing.T) {
	setupTimeTest(t)
	backend := &fakeClockBackend{}
	audits := &recordingAuditLogger{}
	r := newTimeControlRouter(NewTimeControlHandler("catalog", backend, audits))
	target := time.Now().AddDate(1, 0, 0)

	w, body := doJSON(t, r, "POST", "/admin/time/simulate", simulateBody(target))

	require.Equal(t, http.StatusOK, w.Code, body)
	assert.Equal(t, true, body["active"])
	assert.Equal(t, "catalog", body["service"])
	assert.WithinDuration(t, target, clock.Now(), 2*time.Second)

	var stored clocksync.State
	require.NoError(t, json.Unmarshal([]byte(backend.value), &stored))
	assert.InDelta(t, time.Until(target).Seconds(), float64(stored.OffsetSeconds), 2)
	assert.Nil(t, stored.Until, "K1: the simulation does not expire on its own")
	assert.Equal(t, "admin-1", stored.SetBy)

	require.Len(t, audits.events, 1)
	assert.Equal(t, "time.simulated", audits.events[0].Action)
	assert.Equal(t, "admin-1", audits.events[0].ActorID)
	assert.Equal(t, "admin", audits.events[0].ActorRole)
	assert.Equal(t, stored.OffsetSeconds, audits.events[0].Details["offset_seconds"])
}

func TestTimeControl_Simulate_BeyondTwoYears_Returns400(t *testing.T) {
	for name, target := range map[string]time.Time{
		"future": time.Now().AddDate(2, 0, 1),
		"past":   time.Now().AddDate(-2, 0, -1),
	} {
		t.Run(name, func(t *testing.T) {
			setupTimeTest(t)
			backend := &fakeClockBackend{}
			audits := &recordingAuditLogger{}
			r := newTimeControlRouter(NewTimeControlHandler("catalog", backend, audits))

			w, body := doJSON(t, r, "POST", "/admin/time/simulate", simulateBody(target))

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "VALIDATION_ERROR", body["code"])
			assert.Contains(t, body["error"], "2 yıl")
			assert.Empty(t, backend.value)
			assert.False(t, clock.State().Active)
			assert.Empty(t, audits.events)
		})
	}
}

func TestTimeControl_Simulate_WithinTwoYears_Accepted(t *testing.T) {
	setupTimeTest(t)
	r := newTimeControlRouter(NewTimeControlHandler("catalog", &fakeClockBackend{}, nil))

	w, _ := doJSON(t, r, "POST", "/admin/time/simulate", simulateBody(time.Now().AddDate(-2, 0, 1)))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTimeControl_Simulate_InvalidBody_Returns400(t *testing.T) {
	setupTimeTest(t)
	r := newTimeControlRouter(NewTimeControlHandler("catalog", &fakeClockBackend{}, nil))

	for _, body := range []string{`{}`, `{"time":"yarin"}`, `not json`} {
		w, out := doJSON(t, r, "POST", "/admin/time/simulate", body)
		assert.Equal(t, http.StatusBadRequest, w.Code, body)
		assert.Equal(t, "VALIDATION_ERROR", out["code"])
	}
	assert.False(t, clock.State().Active)
}

func TestTimeControl_RedisDown_Returns503AndKeepsClock(t *testing.T) {
	setupTimeTest(t)
	backend := &fakeClockBackend{storeErr: errors.New("connection refused")}
	audits := &recordingAuditLogger{}
	r := newTimeControlRouter(NewTimeControlHandler("catalog", backend, audits))

	w, body := doJSON(t, r, "POST", "/admin/time/simulate", simulateBody(time.Now().Add(time.Hour)))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "SERVICE_UNAVAILABLE", body["code"])
	assert.False(t, clock.State().Active, "moving only this service's clock would split the cluster")

	clock.SetOffset(time.Hour, time.Time{})
	w, _ = doJSON(t, r, "POST", "/admin/time/reset", "")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, clock.State().Active)
	assert.Empty(t, audits.events)
}

func TestTimeControl_NoBackend_Returns503(t *testing.T) {
	setupTimeTest(t)
	r := newTimeControlRouter(NewTimeControlHandler("catalog", nil, nil))

	w, _ := doJSON(t, r, "POST", "/admin/time/simulate", simulateBody(time.Now().Add(time.Hour)))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	w, _ = doJSON(t, r, "POST", "/admin/time/reset", "")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTimeControl_Reset_ClearsAndAudits(t *testing.T) {
	setupTimeTest(t)
	backend := &fakeClockBackend{}
	audits := &recordingAuditLogger{}
	r := newTimeControlRouter(NewTimeControlHandler("catalog", backend, audits))
	w, _ := doJSON(t, r, "POST", "/admin/time/simulate", simulateBody(time.Now().Add(48*time.Hour)))
	require.Equal(t, http.StatusOK, w.Code)

	w, body := doJSON(t, r, "POST", "/admin/time/reset", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, false, body["active"])
	assert.Empty(t, backend.value)
	assert.False(t, clock.State().Active)
	require.Len(t, audits.events, 2)
	assert.Equal(t, "time.reset", audits.events[1].Action)
}
