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

func TestLoad_DemoConfig_DefaultAndOverride(t *testing.T) {
	cfg, err := loadWithEnv(t, nil)
	require.NoError(t, err)
	assert.False(t, cfg.Demo.Enabled)
	assert.Equal(t, "demo.admin@mydreamcampus.com", cfg.Demo.AdminEmail)
	assert.Equal(t, "ahmet.yilmaz@uni.edu.tr", cfg.Demo.TeacherEmail)
	assert.Equal(t, "zeynep.sahin@uni.edu.tr", cfg.Demo.StudentEmail)

	cfg, err = loadWithEnv(t, map[string]string{
		"DEMO_MODE":          "true",
		"DEMO_ADMIN_EMAIL":   "custom.admin@campus.local",
		"DEMO_TEACHER_EMAIL": "custom.teacher@campus.local",
		"DEMO_STUDENT_EMAIL": "custom.student@campus.local",
	})
	require.NoError(t, err)
	assert.True(t, cfg.Demo.Enabled)
	assert.Equal(t, "custom.admin@campus.local", cfg.Demo.AdminEmail)
	assert.Equal(t, "custom.teacher@campus.local", cfg.Demo.TeacherEmail)
	assert.Equal(t, "custom.student@campus.local", cfg.Demo.StudentEmail)
}

func TestLoad_ProtectedAccountEmails_DefaultAndOverride(t *testing.T) {
	cfg, err := loadWithEnv(t, nil)
	require.NoError(t, err)
	expectedDefaults := []string{
		"admin@university.edu.tr",
		"demo.admin@mydreamcampus.com",
		"ahmet.yilmaz@uni.edu.tr",
		"zeynep.sahin@uni.edu.tr",
	}
	assert.Equal(t, expectedDefaults, cfg.ProtectedAccountEmails)

	cfg, err = loadWithEnv(t, map[string]string{
		"PROTECTED_ACCOUNT_EMAILS": "root@uni.edu.tr, vip@uni.edu.tr ",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"root@uni.edu.tr", "vip@uni.edu.tr"}, cfg.ProtectedAccountEmails)
}

func TestConfig_IsProtectedEmail(t *testing.T) {
	cfg, err := loadWithEnv(t, nil)
	require.NoError(t, err)

	// Defaults should be protected (case insensitive)
	assert.True(t, cfg.IsProtectedEmail("ADMIN@university.edu.tr"))
	assert.True(t, cfg.IsProtectedEmail("demo.admin@mydreamcampus.com"))
	assert.True(t, cfg.IsProtectedEmail("ahmet.yilmaz@uni.edu.tr"))
	assert.True(t, cfg.IsProtectedEmail("zeynep.sahin@uni.edu.tr"))

	// Non-protected emails
	assert.False(t, cfg.IsProtectedEmail("random@uni.edu.tr"))
	assert.False(t, cfg.IsProtectedEmail(""))

	// Custom override
	customCfg, err := loadWithEnv(t, map[string]string{
		"ADMIN_EMAIL":              "custom.admin@campus.local",
		"DEMO_ADMIN_EMAIL":         "custom.demo@campus.local",
		"PROTECTED_ACCOUNT_EMAILS": "special@campus.local",
	})
	require.NoError(t, err)
	assert.True(t, customCfg.IsProtectedEmail("custom.admin@campus.local"))
	assert.True(t, customCfg.IsProtectedEmail("custom.demo@campus.local"))
	assert.True(t, customCfg.IsProtectedEmail("special@campus.local"))
	assert.False(t, customCfg.IsProtectedEmail("random@campus.local"))
}
