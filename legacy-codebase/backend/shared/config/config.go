package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

// NOTE: This is a TEMPLATE config package for shared configuration patterns.
// Each microservice should have its own config package with service-specific settings.
// This package demonstrates Viper best practices for the mydreamcampus project.

// BaseConfig represents common configuration shared across all services
type BaseConfig struct {
	Server   ServerConfig
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	Redis    RedisConfig
	JWT      JWTConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Environment string `mapstructure:"ENVIRONMENT"`
	Port        string `mapstructure:"PORT"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	URL string `mapstructure:"DB_URL"`
}

// RabbitMQConfig represents RabbitMQ configuration
type RabbitMQConfig struct {
	URL string `mapstructure:"RABBITMQ_URL"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Addr     string `mapstructure:"REDIS_ADDR"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	Secret string `mapstructure:"JWT_SECRET"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled            bool `mapstructure:"RATE_LIMIT_ENABLED"`
	IPLimit            int  `mapstructure:"RATE_LIMIT_IP_LIMIT"`
	IPWindowSecs       int  `mapstructure:"RATE_LIMIT_IP_WINDOW_SECS"`
	UserLimit          int  `mapstructure:"RATE_LIMIT_USER_LIMIT"`
	UserWindowSecs     int  `mapstructure:"RATE_LIMIT_USER_WINDOW_SECS"`
	LoginLimit         int  `mapstructure:"RATE_LIMIT_LOGIN_LIMIT"`
	LoginWindowSecs    int  `mapstructure:"RATE_LIMIT_LOGIN_WINDOW_SECS"`
	RefreshLimit       int  `mapstructure:"RATE_LIMIT_REFRESH_LIMIT"`
	RefreshWindowSecs  int  `mapstructure:"RATE_LIMIT_REFRESH_WINDOW_SECS"`
	PasswordLimit      int  `mapstructure:"RATE_LIMIT_PASSWORD_LIMIT"`
	PasswordWindowSecs int  `mapstructure:"RATE_LIMIT_PASSWORD_WINDOW_SECS"`
}

// TimeoutConfig represents timeout and duration configuration
type TimeoutConfig struct {
	RequestTimeoutSeconds          int `mapstructure:"REQUEST_TIMEOUT_SECONDS"`
	ShutdownTimeoutSeconds         int `mapstructure:"SHUTDOWN_TIMEOUT_SECONDS"`
	AccountLockDurationMinutes     int `mapstructure:"ACCOUNT_LOCK_DURATION_MINUTES"`
	CleanupSchedulerIntervalHours  int `mapstructure:"CLEANUP_SCHEDULER_INTERVAL_HOURS"`
	ProcessedEventsRetentionDays   int `mapstructure:"PROCESSED_EVENTS_RETENTION_DAYS"`
}

// LoadBaseConfig loads base configuration following Viper best practices
// This is a template function - copy and customize for each service
func LoadBaseConfig(serviceName string, defaultPort string, defaultDBName string) (*BaseConfig, error) {
	// Set default values FIRST
	setBaseDefaults(defaultPort, defaultDBName)

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
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; will use env vars and defaults
	}

	// Build config struct using Viper's type-safe getters
	config := &BaseConfig{
		Server: ServerConfig{
			Environment: viper.GetString("ENVIRONMENT"),
			Port:        viper.GetString("PORT"),
		},
		Database: DatabaseConfig{
			URL: viper.GetString("DB_URL"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: viper.GetString("RABBITMQ_URL"),
		},
		Redis: RedisConfig{
			Addr:     viper.GetString("REDIS_ADDR"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			Secret: viper.GetString("JWT_SECRET"),
		},
	}

	// Validate base config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// setBaseDefaults sets default configuration values
func setBaseDefaults(defaultPort string, defaultDBName string) {
	// Server defaults
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("PORT", defaultPort)

	// Database defaults
	viper.SetDefault("DB_URL", fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable", defaultDBName))

	// RabbitMQ defaults
	viper.SetDefault("RABBITMQ_URL", "amqp://rabbitmq:rabbitmq@localhost:5672/")

	// Redis defaults
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)

	// JWT defaults
	viper.SetDefault("JWT_SECRET", "change-this-secret-in-production")
}

// Validate checks if the base configuration is valid
func (c *BaseConfig) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	if c.Database.URL == "" {
		return fmt.Errorf("DB_URL is required")
	}
	if c.RabbitMQ.URL == "" {
		return fmt.Errorf("RABBITMQ_URL is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.JWT.Secret == "change-this-secret-in-production" && c.Server.Environment == "production" {
		return fmt.Errorf("JWT_SECRET must be changed in production")
	}
	return nil
}