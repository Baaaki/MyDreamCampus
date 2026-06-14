package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheck pings a single dependency and returns nil when healthy.
type HealthCheck func(ctx context.Context) error

// LivenessHandler returns 200 as long as the process can serve HTTP.
// Used by orchestrators to decide when to restart the container.
func LivenessHandler(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "alive",
			"service": serviceName,
		})
	}
}

// ReadinessHandler returns 200 only when every check passes within the
// per-request timeout. Returns 503 with a per-dependency status map otherwise.
// Used by orchestrators to decide when to send traffic.
func ReadinessHandler(serviceName string, checks map[string]HealthCheck) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		results := make(map[string]string, len(checks))
		ready := true
		for name, check := range checks {
			if err := check(ctx); err != nil {
				results[name] = "down: " + err.Error()
				ready = false
			} else {
				results[name] = "up"
			}
		}

		status := http.StatusOK
		state := "ready"
		if !ready {
			status = http.StatusServiceUnavailable
			state = "not_ready"
		}

		c.JSON(status, gin.H{
			"status":  state,
			"service": serviceName,
			"checks":  results,
		})
	}
}
