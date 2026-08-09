package client

import (
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/shared/config"
)

// FromConfig builds the transport for one target service. Every caller goes
// through here so the timeout, the internal secret, the request-id hand-off
// and the breaker thresholds are identical on all seven sync seams.
//
// The base URL comes from config — never from request data — so a caller
// cannot be talked into pointing an internal client somewhere else (SSRF).
func FromConfig(cfg *config.Config, target string) (*Base, error) {
	baseURL := cfg.InternalClient.ServiceURLs[target]
	if baseURL == "" {
		return nil, fmt.Errorf("%s service URL is not configured", target)
	}

	breaker := cfg.InternalClient.Breaker
	return NewBase(Config{
		Target:  target,
		BaseURL: baseURL,
		Secret:  cfg.Server.InternalSecret,
		Timeout: time.Duration(cfg.InternalClient.TimeoutSeconds) * time.Second,
		Breaker: BreakerConfig{
			MaxRequests:         uint32(max(breaker.MaxRequests, 0)),
			Timeout:             time.Duration(breaker.TimeoutSeconds) * time.Second,
			ConsecutiveFailures: uint32(max(breaker.ConsecutiveFailures, 0)),
		},
	}), nil
}
