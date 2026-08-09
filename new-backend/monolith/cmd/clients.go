package main

import (
	"time"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/client"
)

// internalTransports holds one shared/client.Base per target service. The
// breaker inside each one is scoped to that target, so a dead staff service
// does not cut off calls to catalog.
type internalTransports struct {
	staff   *client.Base
	student *client.Base
	catalog *client.Base
	payment *client.Base
	meal    *client.Base
}

// orInProcess returns the HTTP client when one was built, otherwise the
// in-process adapter. The adapter is built lazily because it needs a module
// handle that only exists in this mode.
func orInProcess[T comparable](httpClient T, inProcess func() T) T {
	var zero T
	if httpClient != zero {
		return httpClient
	}
	return inProcess()
}

// loopbackPrefixes maps a target to the route prefix it answers on inside
// the monolith. Once the services are split, each gets its own host and the
// prefix disappears — the paths the clients use are already the final ones.
var loopbackPrefixes = map[string]string{
	"staff":   "/api/staff",
	"student": "/api/students",
	"catalog": "/api/catalog",
	"payment": "/api/payments",
	"meal":    "/api/meals",
}

func newInternalTransports(cfg *config.Config) *internalTransports {
	return &internalTransports{
		staff:   newTransport(cfg, "staff"),
		student: newTransport(cfg, "student"),
		catalog: newTransport(cfg, "catalog"),
		payment: newTransport(cfg, "payment"),
		meal:    newTransport(cfg, "meal"),
	}
}

func newTransport(cfg *config.Config, target string) *client.Base {
	baseURL := cfg.InternalClient.ServiceURLs[target]
	if baseURL == "" {
		baseURL = "http://localhost:" + cfg.Server.Port + loopbackPrefixes[target]
	}

	breaker := cfg.InternalClient.Breaker
	return client.NewBase(client.Config{
		Target:  target,
		BaseURL: baseURL,
		Secret:  cfg.Server.InternalSecret,
		Timeout: time.Duration(cfg.InternalClient.TimeoutSeconds) * time.Second,
		Breaker: client.BreakerConfig{
			MaxRequests:         uint32(max(breaker.MaxRequests, 0)),
			Timeout:             time.Duration(breaker.TimeoutSeconds) * time.Second,
			ConsecutiveFailures: uint32(max(breaker.ConsecutiveFailures, 0)),
		},
	})
}
