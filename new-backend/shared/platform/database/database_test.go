package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDSN = "postgres://svc:secret@localhost:5432/app?sslmode=disable"

func TestPoolConfig_NoLimit_UsesDefault(t *testing.T) {
	cfg, err := poolConfig(testDSN, 0)
	require.NoError(t, err)

	assert.EqualValues(t, defaultMaxConns, cfg.MaxConns)
	assert.EqualValues(t, 2, cfg.MinConns)
}

func TestPoolConfig_ExplicitLimit_IsApplied(t *testing.T) {
	cfg, err := poolConfig(testDSN, 4)
	require.NoError(t, err)

	assert.EqualValues(t, 4, cfg.MaxConns)
	assert.EqualValues(t, 2, cfg.MinConns)
}

func TestPoolConfig_LimitOfOne_MinConnsNotAboveMax(t *testing.T) {
	cfg, err := poolConfig(testDSN, 1)
	require.NoError(t, err)

	assert.EqualValues(t, 1, cfg.MaxConns)
	assert.EqualValues(t, 1, cfg.MinConns)
}
