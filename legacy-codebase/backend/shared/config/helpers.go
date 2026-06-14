package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// SetupViper initializes Viper with standard settings for all services
// This should be called before loading config values
// serviceName: e.g., "auth-service", "staff-service", "student-service"
func SetupViper(serviceName string) error {
	// Set config name and type
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	// Add config paths (search order)
	viper.AddConfigPath(fmt.Sprintf("/etc/%s/", serviceName))
	viper.AddConfigPath(fmt.Sprintf("$HOME/.%s", serviceName))
	viper.AddConfigPath(".")

	// Read from environment variables (overrides config file)
	viper.AutomaticEnv()

	// Read config file (optional, env vars take priority)
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			// Config file was found but another error was produced
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; will use env vars and defaults
	}

	return nil
}

// SetCommonDefaults sets default values common to all services
// serviceName: e.g., "auth", "staff", "student" (without -service suffix)
// defaultPort: e.g., "8001", "8002", "8003"
// dbPort: database port, e.g., "5432", "5433", "5434"
func SetCommonDefaults(serviceName, defaultPort, dbPort string) {
	// Server defaults
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("PORT", defaultPort)

	// Database defaults
	viper.SetDefault("DB_URL", fmt.Sprintf(
		"postgres://postgres:postgres@localhost:%s/mydreamcampus_%s?sslmode=disable",
		dbPort,
		serviceName,
	))

	// RabbitMQ defaults
	viper.SetDefault("RABBITMQ_URL", "amqp://rabbitmq:rabbitmq@localhost:5672/")

	// Redis defaults
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "changeme_redis_secret")
	viper.SetDefault("REDIS_DB", 0)

	// JWT defaults
	viper.SetDefault("JWT_SECRET", "change-this-secret-in-production")

	// INTERNAL_SERVICE_SECRET intentionally has no default — must be provided via env.

	// Rate limiting defaults
	viper.SetDefault("RATE_LIMIT_ENABLED", true)
	viper.SetDefault("RATE_LIMIT_IP_LIMIT", 100)
	viper.SetDefault("RATE_LIMIT_IP_WINDOW_SECS", 60)
	viper.SetDefault("RATE_LIMIT_USER_LIMIT", 200)
	viper.SetDefault("RATE_LIMIT_USER_WINDOW_SECS", 60)
	viper.SetDefault("RATE_LIMIT_LOGIN_LIMIT", 20)
	viper.SetDefault("RATE_LIMIT_LOGIN_WINDOW_SECS", 60)
	viper.SetDefault("RATE_LIMIT_REFRESH_LIMIT", 10)
	viper.SetDefault("RATE_LIMIT_REFRESH_WINDOW_SECS", 60)
	viper.SetDefault("RATE_LIMIT_PASSWORD_LIMIT", 3)
	viper.SetDefault("RATE_LIMIT_PASSWORD_WINDOW_SECS", 300)

	// Timeout defaults (in seconds)
	viper.SetDefault("REQUEST_TIMEOUT_SECONDS", 10)
	viper.SetDefault("SHUTDOWN_TIMEOUT_SECONDS", 30)
	viper.SetDefault("ACCOUNT_LOCK_DURATION_MINUTES", 30)
	viper.SetDefault("CLEANUP_SCHEDULER_INTERVAL_HOURS", 1)
	viper.SetDefault("PROCESSED_EVENTS_RETENTION_DAYS", 30)
}

// LoadCommonConfig loads common configuration fields shared across all services
// Returns the common config structs that all services use
func LoadCommonConfig() (ServerConfig, DatabaseConfig, RabbitMQConfig, RedisConfig, JWTConfig) {
	server := ServerConfig{
		Environment: viper.GetString("ENVIRONMENT"),
		Port:        viper.GetString("PORT"),
	}

	database := DatabaseConfig{
		URL: viper.GetString("DB_URL"),
	}

	rabbitmq := RabbitMQConfig{
		URL: viper.GetString("RABBITMQ_URL"),
	}

	redis := RedisConfig{
		Addr:     viper.GetString("REDIS_ADDR"),
		Password: viper.GetString("REDIS_PASSWORD"),
		DB:       viper.GetInt("REDIS_DB"),
	}

	jwt := JWTConfig{
		Secret: viper.GetString("JWT_SECRET"),
	}

	// LEGACY SUPPORT: Set env var for utils.GetJWTSecret()
	// This ensures that legacy code relying on os.Getenv("JWT_SECRET") works
	// even if the value came from a config file via Viper.
	if jwt.Secret != "" {
		os.Setenv("JWT_SECRET", jwt.Secret)
	}

	return server, database, rabbitmq, redis, jwt
}

// LoadRateLimitConfig loads rate limiting configuration from Viper.
// This is a separate function to avoid breaking the LoadCommonConfig signature.
func LoadRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:            viper.GetBool("RATE_LIMIT_ENABLED"),
		IPLimit:            viper.GetInt("RATE_LIMIT_IP_LIMIT"),
		IPWindowSecs:       viper.GetInt("RATE_LIMIT_IP_WINDOW_SECS"),
		UserLimit:          viper.GetInt("RATE_LIMIT_USER_LIMIT"),
		UserWindowSecs:     viper.GetInt("RATE_LIMIT_USER_WINDOW_SECS"),
		LoginLimit:         viper.GetInt("RATE_LIMIT_LOGIN_LIMIT"),
		LoginWindowSecs:    viper.GetInt("RATE_LIMIT_LOGIN_WINDOW_SECS"),
		RefreshLimit:       viper.GetInt("RATE_LIMIT_REFRESH_LIMIT"),
		RefreshWindowSecs:  viper.GetInt("RATE_LIMIT_REFRESH_WINDOW_SECS"),
		PasswordLimit:      viper.GetInt("RATE_LIMIT_PASSWORD_LIMIT"),
		PasswordWindowSecs: viper.GetInt("RATE_LIMIT_PASSWORD_WINDOW_SECS"),
	}
}

// minJWTSecretLength is the minimum acceptable length in bytes for a
// production JWT_SECRET. HS256 derives an HMAC key directly from this
// value; anything shorter than 32 bytes weakens the signing strength
// and makes brute-force feasible.
const minJWTSecretLength = 32

// ValidateCommonConfig validates common configuration fields
// Returns error if any required field is missing or invalid
func ValidateCommonConfig(server ServerConfig, database DatabaseConfig, rabbitmq RabbitMQConfig, jwt JWTConfig) error {
	if server.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	if database.URL == "" {
		return fmt.Errorf("DB_URL is required")
	}
	if rabbitmq.URL == "" {
		return fmt.Errorf("RABBITMQ_URL is required")
	}
	if jwt.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if jwt.Secret == "change-this-secret-in-production" && server.Environment == "production" {
		return fmt.Errorf("JWT_SECRET must be changed in production")
	}
	if server.Environment == "production" && len(jwt.Secret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d bytes in production (got %d)", minJWTSecretLength, len(jwt.Secret))
	}
	return nil
}

// ValidateRedisConfig enforces that REDIS_ADDR is set when a service
// depends on Redis (rate limiting, token blacklist, session storage).
// In production we additionally require a real REDIS_PASSWORD so the
// cache cannot be reached unauthenticated even if it ends up exposed.
func ValidateRedisConfig(server ServerConfig, redis RedisConfig) error {
	if redis.Addr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}
	if server.Environment == "production" {
		if redis.Password == "" {
			return fmt.Errorf("REDIS_PASSWORD is required in production")
		}
		if redis.Password == "changeme_redis_secret" {
			return fmt.Errorf("REDIS_PASSWORD must be changed in production")
		}
	}
	return nil
}
