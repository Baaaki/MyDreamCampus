package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// IdempotencyHeader is the client-generated key (UUID v4) that ties a retry
// to its original attempt.
const IdempotencyHeader = "Idempotency-Key"

// idempotencyTTL is how long a completed response stays replayable. A day
// covers a mobile client that retries after a long offline stretch.
const idempotencyTTL = 24 * time.Hour

// inFlightTTL bounds the "still processing" claim. It only has to outlive
// one request; if the process dies mid-request nothing releases the claim,
// and with the full day's TTL the key would answer 409 until tomorrow.
const inFlightTTL = 2 * time.Minute

// idempotencyWriteTimeout bounds the Save/Release that runs after the
// handler, on a context that no longer follows the client.
const idempotencyWriteTimeout = 3 * time.Second

// maxIdempotentBodyBytes caps what we buffer to hash and store. Requests
// above it skip idempotency rather than pull an unbounded body into memory.
const maxIdempotentBodyBytes = 1 << 20 // 1 MiB

// IdempotencyStore is the state the middleware needs. Implemented by
// redis.ClientWrapper.
type IdempotencyStore interface {
	GetIdempotencyRecord(ctx context.Context, key string) (string, error)
	ClaimIdempotencyKey(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	SaveIdempotencyRecord(ctx context.Context, key, value string, ttl time.Duration) error
	ReleaseIdempotencyKey(ctx context.Context, key string) error
}

// idempotencyRecord is what a key maps to. An in-flight claim carries the
// body hash only; StatusCode is filled in once the handler has answered.
type idempotencyRecord struct {
	BodySHA256   string `json:"body_sha256"`
	StatusCode   int    `json:"status_code,omitempty"`
	ResponseBody string `json:"response_body,omitempty"`
	InFlight     bool   `json:"in_flight"`
}

// idempotencyConfig is set once at startup; nil means the feature is off
// and every request passes straight through.
type idempotencyConfig struct {
	store       IdempotencyStore
	serviceName string
}

var globalIdempotency *idempotencyConfig

// SetIdempotencyStore wires the middleware during service startup. The
// service name namespaces the keys so two services cannot collide on a key
// a client reused.
func SetIdempotencyStore(store IdempotencyStore, serviceName string) {
	globalIdempotency = &idempotencyConfig{store: store, serviceName: serviceName}
}

// Idempotency makes a mutating endpoint safe to retry: the same
// Idempotency-Key replays the first response instead of performing the work
// twice. Mount it on the routes that move money or consume quota, not
// globally — every request would pay a Redis round-trip for nothing.
//
// Redis being down fails OPEN, unlike the rate limiter. This is duplicate
// protection, not an access control; refusing reservations because Redis
// blinked is the worse outcome.
func Idempotency() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := globalIdempotency
		if cfg == nil {
			c.Next()
			return
		}
		// DELETE is idempotent on the resource but not on its side effects:
		// cancelling a reservation twice must not refund it twice.
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			c.Next()
			return
		}

		clientKey := c.GetHeader(IdempotencyHeader)
		if clientKey == "" {
			c.Next() // opt-in: clients that do not send the header are unaffected
			return
		}

		body, ok := readBodyForIdempotency(c)
		if !ok {
			c.Next()
			return
		}

		key := idempotencyKey(cfg.serviceName, userIDForIdempotency(c), clientKey)
		bodyHash := sha256Hex(body)
		ctx := c.Request.Context()
		log := logger.WithContextAndFields(ctx,
			zap.String("middleware", "Idempotency"),
			zap.String("idempotency_key", clientKey),
		)

		existing, err := cfg.store.GetIdempotencyRecord(ctx, key)
		if err != nil {
			log.Warn("idempotency lookup failed, passing request through", zap.Error(err))
			c.Next()
			return
		}
		if existing != "" {
			replayIdempotentResponse(c, log, existing, bodyHash)
			return
		}

		claim, err := json.Marshal(idempotencyRecord{BodySHA256: bodyHash, InFlight: true})
		if err != nil {
			log.Warn("idempotency claim could not be encoded", zap.Error(err))
			c.Next()
			return
		}
		claimed, err := cfg.store.ClaimIdempotencyKey(ctx, key, string(claim), inFlightTTL)
		if err != nil {
			log.Warn("idempotency claim failed, passing request through", zap.Error(err))
			c.Next()
			return
		}
		if !claimed {
			// Someone won the race between our GET and this SETNX.
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": "İşleminiz sürüyor, lütfen bekleyin",
				"code":  "IDEMPOTENCY_IN_PROGRESS",
			})
			return
		}

		recorder := &idempotencyRecorder{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = recorder

		c.Next()

		// The work is done whether or not the client stayed to hear about it.
		// On the request context a client that hung up — the very one that
		// will retry — would cancel the Save, leaving the claim stuck.
		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), idempotencyWriteTimeout)
		defer cancel()

		status := recorder.Status()
		// Only successful work is worth replaying. Releasing the key on
		// failure lets the client retry the same key right away.
		if status < 200 || status >= 300 {
			if err := cfg.store.ReleaseIdempotencyKey(writeCtx, key); err != nil {
				log.Warn("failed to release idempotency key after error response", zap.Error(err))
			}
			return
		}

		record, err := json.Marshal(idempotencyRecord{
			BodySHA256:   bodyHash,
			StatusCode:   status,
			ResponseBody: recorder.body.String(),
		})
		if err != nil {
			log.Warn("idempotent response could not be encoded", zap.Error(err))
			return
		}
		if err := cfg.store.SaveIdempotencyRecord(writeCtx, key, string(record), idempotencyTTL); err != nil {
			log.Warn("failed to store idempotent response", zap.Error(err))
		}
	}
}

// replayIdempotentResponse answers from the stored record: the same body
// for a repeat of the same request, 409 while the first one is still
// running, 422 when the key was reused for different content.
func replayIdempotentResponse(c *gin.Context, log *zap.Logger, stored, bodyHash string) {
	var record idempotencyRecord
	if err := json.Unmarshal([]byte(stored), &record); err != nil {
		log.Warn("stored idempotency record is unreadable, passing request through", zap.Error(err))
		c.Next()
		return
	}

	if record.BodySHA256 != bodyHash {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"error": "Bu anahtar farklı bir istekle kullanıldı",
			"code":  "IDEMPOTENCY_KEY_REUSED",
		})
		return
	}
	if record.InFlight {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"error": "İşleminiz sürüyor, lütfen bekleyin",
			"code":  "IDEMPOTENCY_IN_PROGRESS",
		})
		return
	}

	c.Abort()
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(record.StatusCode, "%s", record.ResponseBody)
}

// readBodyForIdempotency buffers the body so it can be hashed and still
// reach the handler. Oversized bodies opt out instead of being buffered.
func readBodyForIdempotency(c *gin.Context) ([]byte, bool) {
	if c.Request.Body == nil {
		return nil, true
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxIdempotentBodyBytes+1))
	if err != nil || len(body) > maxIdempotentBodyBytes {
		c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), c.Request.Body))
		return nil, false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, true
}

// userIDForIdempotency scopes the key to the caller so one user's key can
// never read another user's stored response.
func userIDForIdempotency(c *gin.Context) string {
	if userID, ok := c.Get("user_id"); ok {
		if s, ok := userID.(string); ok && s != "" {
			return s
		}
	}
	return "anonymous"
}

func idempotencyKey(service, userID, clientKey string) string {
	return "idem:" + service + ":" + userID + ":" + clientKey
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// idempotencyRecorder captures the response so it can be replayed later.
type idempotencyRecorder struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *idempotencyRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *idempotencyRecorder) WriteString(s string) (int, error) {
	r.body.WriteString(s)
	return r.ResponseWriter.WriteString(s)
}
