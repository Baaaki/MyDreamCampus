// Package client carries the internal REST calls services make to each
// other. Every request leaves with the shared X-Internal-Secret; the
// receiving side verifies it with platform/middleware.RequireInternalSecret.
//
// One Base instance per target service: the circuit breaker it holds is
// scoped to that target, so a dead staff service must not cut off calls
// going to catalog.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/sony/gobreaker/v2"
	"go.uber.org/zap"
)

// ErrNotFound is returned for a 404 from the target service. Callers map it
// onto their own module sentinel so handlers keep producing the same status
// code they did when the call was an in-process function call.
var ErrNotFound = errors.New("internal client: resource not found")

// ErrUnavailable is in the chain of every error that means "the target could
// not answer" — transport failure, 5xx, or an open breaker. Callers that
// degrade gracefully on a missing record must NOT degrade on this one: a
// check that cannot be performed is not a check that passed.
var ErrUnavailable = errors.New("internal client: target service unavailable")

// errServerSide marks a 5xx so the breaker counts it. It never leaves this
// package — callers see the StatusError instead.
var errServerSide = errors.New("internal client: server-side failure")

// maxErrorBodyBytes caps how much of a failed response we read into the
// error message; the body is for diagnosis, not for parsing.
const maxErrorBodyBytes = 1024

const (
	defaultTimeout             = 10 * time.Second
	defaultBreakerTimeout      = 30 * time.Second
	defaultBreakerMaxRequests  = 1
	defaultConsecutiveFailures = 5
)

// StatusError carries a non-2xx response the caller has to decide about.
// 404 arrives as ErrNotFound instead — that one is a business answer, not
// a transport failure.
type StatusError struct {
	StatusCode int
	Body       string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("internal client: unexpected status %d: %s", e.StatusCode, e.Body)
}

// Unwrap puts ErrUnavailable in the chain of a 5xx so callers can tell "the
// target broke" from "the target said no".
func (e *StatusError) Unwrap() error {
	if e.StatusCode >= 500 {
		return ErrUnavailable
	}
	return nil
}

// BreakerConfig holds the thresholds the circuit breaker trips on. Values
// come from config (CIRCUIT_BREAKER_*) so a slow homeserver can be tuned
// without a rebuild; zero values fall back to the defaults above.
type BreakerConfig struct {
	// MaxRequests is how many probes the half-open state lets through.
	MaxRequests uint32
	// Timeout is how long the breaker stays open before probing again.
	Timeout time.Duration
	// ConsecutiveFailures is the failure streak that opens the breaker.
	ConsecutiveFailures uint32
}

// Config describes one target service.
type Config struct {
	// Target names the service for logs and breaker state changes.
	Target  string
	BaseURL string
	Secret  string
	Timeout time.Duration
	Breaker BreakerConfig
}

// Base is the shared transport for internal service-to-service calls.
type Base struct {
	target  string
	baseURL string
	secret  string
	http    *http.Client
	breaker *gobreaker.CircuitBreaker[*http.Response]
}

// NewBase builds the transport for one target service.
func NewBase(cfg Config) *Base {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Base{
		target:  cfg.Target,
		baseURL: strings.TrimSuffix(cfg.BaseURL, "/"),
		secret:  cfg.Secret,
		http:    &http.Client{Timeout: timeout},
		breaker: newBreaker(cfg.Target, cfg.Breaker),
	}
}

func newBreaker(target string, cfg BreakerConfig) *gobreaker.CircuitBreaker[*http.Response] {
	maxRequests := cfg.MaxRequests
	if maxRequests == 0 {
		maxRequests = defaultBreakerMaxRequests
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultBreakerTimeout
	}
	failures := cfg.ConsecutiveFailures
	if failures == 0 {
		failures = defaultConsecutiveFailures
	}

	return gobreaker.NewCircuitBreaker[*http.Response](gobreaker.Settings{
		Name:        target,
		MaxRequests: maxRequests,
		Timeout:     timeout,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= failures
		},
		// A breaker that opens silently is one of the hardest failures to
		// diagnose — every transition gets a log line.
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Warn("circuit breaker state change",
				zap.String("target", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	})
}

// Get issues a GET and decodes a 2xx body into out. path must already be
// URL-encoded; never build it from request data (SSRF).
func (b *Base) Get(ctx context.Context, path string, out any) error {
	return b.do(ctx, http.MethodGet, path, nil, out)
}

// Post sends in as JSON and decodes the 2xx body into out. Either may be nil.
func (b *Base) Post(ctx context.Context, path string, in, out any) error {
	return b.do(ctx, http.MethodPost, path, in, out)
}

// Put sends in as JSON and decodes the 2xx body into out. Either may be nil.
func (b *Base) Put(ctx context.Context, path string, in, out any) error {
	return b.do(ctx, http.MethodPut, path, in, out)
}

// Delete issues a DELETE; a 2xx body, if any, is discarded.
func (b *Base) Delete(ctx context.Context, path string) error {
	return b.do(ctx, http.MethodDelete, path, nil, nil)
}

func (b *Base) do(ctx context.Context, method, path string, in, out any) error {
	if b.baseURL == "" {
		return fmt.Errorf("internal client: base URL not configured for %s", b.target)
	}

	var body []byte
	if in != nil {
		encoded, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("internal client: marshal request: %w", err)
		}
		body = encoded
	}

	resp, err := b.breaker.Execute(func() (*http.Response, error) {
		return b.send(ctx, method, path, body)
	})
	if err != nil && !errors.Is(err, errServerSide) {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			logger.WithContext(ctx).Warn("internal call rejected by open circuit breaker",
				zap.String("target", b.target),
				zap.String("path", path),
			)
			return sharedErrors.WrapWithMessage(sharedErrors.ErrServiceUnavailable,
				fmt.Errorf("%w: %w", ErrUnavailable, err),
				"Servis şu anda yanıt vermiyor, lütfen birazdan tekrar deneyin")
		}
		return fmt.Errorf("internal client: %s %s: %w: %w", method, path, ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		return &StatusError{StatusCode: resp.StatusCode, Body: string(errBody)}
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("internal client: decode response from %s: %w", b.target, err)
	}
	return nil
}

// send performs the request and reports 5xx as an error so the breaker
// counts it. 4xx comes back with a nil error on purpose: "student not
// found" is a valid business answer and must not open the breaker.
func (b *Base) send(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, b.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Internal-Secret", b.secret)
	// Joins the log lines this request produces in the callee to the ones it
	// produced here; without it the chain breaks at the service boundary.
	if rid := logger.GetRequestID(ctx); rid != "" {
		req.Header.Set("X-Request-ID", rid)
	}

	resp, err := b.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 500 {
		return resp, errServerSide
	}
	return resp, nil
}
