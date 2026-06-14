package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetViper() {
	viper.Reset()
}

func TestSetCommonDefaults_PopulatesViper(t *testing.T) {
	resetViper()
	defer resetViper()

	SetCommonDefaults("auth", "8001", "5432")

	assert.Equal(t, "development", viper.GetString("ENVIRONMENT"))
	assert.Equal(t, "8001", viper.GetString("PORT"))
	assert.Contains(t, viper.GetString("DB_URL"), "mydreamcampus_auth")
	assert.Contains(t, viper.GetString("DB_URL"), ":5432/")
	assert.Equal(t, "amqp://rabbitmq:rabbitmq@localhost:5672/", viper.GetString("RABBITMQ_URL"))
	assert.Equal(t, "localhost:6379", viper.GetString("REDIS_ADDR"))
	assert.Equal(t, 0, viper.GetInt("REDIS_DB"))
	assert.True(t, viper.GetBool("RATE_LIMIT_ENABLED"))
	assert.Equal(t, 30, viper.GetInt("SHUTDOWN_TIMEOUT_SECONDS"))
}

func TestSetCommonDefaults_NoDefaultForInternalSecret(t *testing.T) {
	resetViper()
	defer resetViper()
	SetCommonDefaults("auth", "8001", "5432")

	// INTERNAL_SERVICE_SECRET intentionally has no default — must come from env.
	assert.Empty(t, viper.GetString("INTERNAL_SERVICE_SECRET"))
}

func TestLoadCommonConfig_ReadsViper(t *testing.T) {
	resetViper()
	defer resetViper()
	SetCommonDefaults("auth", "8001", "5432")
	viper.Set("JWT_SECRET", "test-secret-32-chars-minimum-required")

	server, db, rabbit, redis, jwt := LoadCommonConfig()

	assert.Equal(t, "8001", server.Port)
	assert.Equal(t, "development", server.Environment)
	assert.Contains(t, db.URL, "mydreamcampus_auth")
	assert.NotEmpty(t, rabbit.URL)
	assert.Equal(t, "localhost:6379", redis.Addr)
	assert.Equal(t, "test-secret-32-chars-minimum-required", jwt.Secret)
}

func TestLoadRateLimitConfig(t *testing.T) {
	resetViper()
	defer resetViper()
	SetCommonDefaults("auth", "8001", "5432")

	rl := LoadRateLimitConfig()
	assert.True(t, rl.Enabled)
	assert.Equal(t, 100, rl.IPLimit)
	assert.Equal(t, 60, rl.IPWindowSecs)
	assert.Equal(t, 200, rl.UserLimit)
	assert.Equal(t, 20, rl.LoginLimit)
	assert.Equal(t, 3, rl.PasswordLimit)
}

func TestValidateCommonConfig(t *testing.T) {
	makeValid := func() (ServerConfig, DatabaseConfig, RabbitMQConfig, JWTConfig) {
		return ServerConfig{Environment: "development", Port: "8001"},
			DatabaseConfig{URL: "postgres://x"},
			RabbitMQConfig{URL: "amqp://x"},
			JWTConfig{Secret: "any-dev-secret"}
	}

	t.Run("valid dev config passes", func(t *testing.T) {
		s, d, r, j := makeValid()
		assert.NoError(t, ValidateCommonConfig(s, d, r, j))
	})

	t.Run("missing port fails", func(t *testing.T) {
		s, d, r, j := makeValid()
		s.Port = ""
		err := ValidateCommonConfig(s, d, r, j)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PORT")
	})

	t.Run("missing db url fails", func(t *testing.T) {
		s, d, r, j := makeValid()
		d.URL = ""
		err := ValidateCommonConfig(s, d, r, j)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_URL")
	})

	t.Run("missing rabbitmq fails", func(t *testing.T) {
		s, d, r, j := makeValid()
		r.URL = ""
		err := ValidateCommonConfig(s, d, r, j)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "RABBITMQ_URL")
	})

	t.Run("missing jwt fails", func(t *testing.T) {
		s, d, r, j := makeValid()
		j.Secret = ""
		err := ValidateCommonConfig(s, d, r, j)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_SECRET")
	})

	t.Run("default jwt rejected in production", func(t *testing.T) {
		s, d, r, j := makeValid()
		s.Environment = "production"
		j.Secret = "change-this-secret-in-production"
		err := ValidateCommonConfig(s, d, r, j)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be changed")
	})

	t.Run("short jwt rejected in production", func(t *testing.T) {
		s, d, r, j := makeValid()
		s.Environment = "production"
		j.Secret = "short" // < 32 bytes
		err := ValidateCommonConfig(s, d, r, j)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "32 bytes")
	})

	t.Run("32-byte jwt accepted in production", func(t *testing.T) {
		s, d, r, j := makeValid()
		s.Environment = "production"
		j.Secret = strings.Repeat("a", 32)
		assert.NoError(t, ValidateCommonConfig(s, d, r, j))
	})
}

func TestValidateRedisConfig(t *testing.T) {
	t.Run("missing addr fails", func(t *testing.T) {
		err := ValidateRedisConfig(ServerConfig{}, RedisConfig{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "REDIS_ADDR")
	})

	t.Run("dev allows empty password", func(t *testing.T) {
		s := ServerConfig{Environment: "development"}
		r := RedisConfig{Addr: "localhost:6379"}
		assert.NoError(t, ValidateRedisConfig(s, r))
	})

	t.Run("prod requires password", func(t *testing.T) {
		s := ServerConfig{Environment: "production"}
		r := RedisConfig{Addr: "redis:6379"}
		err := ValidateRedisConfig(s, r)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "REDIS_PASSWORD")
	})

	t.Run("prod rejects default password", func(t *testing.T) {
		s := ServerConfig{Environment: "production"}
		r := RedisConfig{Addr: "redis:6379", Password: "changeme_redis_secret"}
		err := ValidateRedisConfig(s, r)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be changed")
	})

	t.Run("prod with real password passes", func(t *testing.T) {
		s := ServerConfig{Environment: "production"}
		r := RedisConfig{Addr: "redis:6379", Password: "actual-strong-password"}
		assert.NoError(t, ValidateRedisConfig(s, r))
	})
}
