package httpserver

import (
	"net/http/httptest"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clientIPSeen(t *testing.T, remoteAddr, forwardedFor string) string {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	cfg := &config.Config{}
	cfg.Server.TrustedProxies = []string{"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}

	s := NewServer(cfg, "test")
	var seen string
	s.Engine().GET("/ip", func(c *gin.Context) { seen = c.ClientIP() })

	req := httptest.NewRequest("GET", "/ip", nil)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	s.Engine().ServeHTTP(httptest.NewRecorder(), req)
	return seen
}

func TestClientIP_DirectPublicPeer_IgnoresForwardedFor(t *testing.T) {
	assert.Equal(t, "198.51.100.20", clientIPSeen(t, "198.51.100.20:4000", "1.2.3.4"),
		"a header written by the client itself must not choose the rate-limit bucket")
}

func TestClientIP_BehindCaddy_UsesForwardedClient(t *testing.T) {
	assert.Equal(t, "198.51.100.7", clientIPSeen(t, "172.18.0.5:4000", "198.51.100.7"))
}

func TestClientIP_BehindCaddy_SpoofedPrefixIgnored(t *testing.T) {
	// Caddy appends the peer it saw; anything left of that came from the client.
	assert.Equal(t, "198.51.100.7", clientIPSeen(t, "172.18.0.5:4000", "6.6.6.6, 198.51.100.7"))
}
