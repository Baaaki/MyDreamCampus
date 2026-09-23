package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadWithEnv(t *testing.T, env map[string]string) (*Config, error) {
	t.Helper()
	viper.Reset()
	t.Chdir(t.TempDir()) // no stray .env from the working directory
	for k, v := range env {
		t.Setenv(k, v)
	}
	return Load()
}

func TestLoad_ReadsQRAndMealTimeSections(t *testing.T) {
	cfg, err := loadWithEnv(t, map[string]string{
		"QR_SECRET":        "qr-secret-from-env",
		"LUNCH_START_HOUR": "12",
	})
	require.NoError(t, err)

	assert.Equal(t, "qr-secret-from-env", cfg.QR.Secret)
	assert.Equal(t, 12, cfg.MealTime.LunchStartHour)
	assert.Equal(t, 13, cfg.MealTime.LunchEndHour, "default must survive too")
	assert.Equal(t, 16, cfg.MealTime.DinnerStartHour)
	assert.Equal(t, 19, cfg.MealTime.DinnerEndHour)
}

func productionEnv(overrides map[string]string) map[string]string {
	env := map[string]string{
		"ENVIRONMENT":             "production",
		"JWT_SECRET":              strings.Repeat("j", 40),
		"REDIS_PASSWORD":          "real-redis-password",
		"ADMIN_INITIAL_PASSWORD":  "Real-admin-1",
		"INTERNAL_SERVICE_SECRET": strings.Repeat("i", 40),
	}
	for k, v := range overrides {
		env[k] = v
	}
	return env
}

func TestLoad_Production_WithoutQRSecret_StillStarts(t *testing.T) {
	// Seven services never get QR_SECRET; Load must not refuse them.
	_, err := loadWithEnv(t, productionEnv(nil))
	assert.NoError(t, err)
}

func TestValidateQRSecret_Production(t *testing.T) {
	cases := map[string]struct {
		secret  string
		wantErr bool
	}{
		"default":   {"change-this-qr-secret-in-production", true},
		"too short": {"short", true},
		"real":      {strings.Repeat("q", 40), false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cfg, err := loadWithEnv(t, productionEnv(map[string]string{"QR_SECRET": c.secret}))
			require.NoError(t, err)
			if c.wantErr {
				assert.ErrorContains(t, cfg.ValidateQRSecret(), "QR_SECRET")
			} else {
				assert.NoError(t, cfg.ValidateQRSecret())
			}
		})
	}
}

func TestValidateQRSecret_Development_AllowsDefault(t *testing.T) {
	cfg, err := loadWithEnv(t, nil)
	require.NoError(t, err)
	assert.NoError(t, cfg.ValidateQRSecret())
}

func TestLoad_DBMaxConns_DefaultAndOverride(t *testing.T) {
	cfg, err := loadWithEnv(t, nil)
	require.NoError(t, err)
	assert.Equal(t, 10, cfg.Database.MaxConns)

	cfg, err = loadWithEnv(t, map[string]string{"DB_MAX_CONNS": "6"})
	require.NoError(t, err)
	assert.Equal(t, 6, cfg.Database.MaxConns)
}
