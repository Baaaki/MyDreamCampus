package config

import (
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/spf13/viper"
)

// Config holds all configuration for the attendance service
type Config struct {
	Server         config.ServerConfig
	Database       config.DatabaseConfig
	RabbitMQ       config.RabbitMQConfig
	Redis          config.RedisConfig
	JWT            config.JWTConfig
	RateLimit      config.RateLimitConfig
	CatalogService CatalogServiceConfig
}

// CatalogServiceConfig holds configuration for Catalog Service integration
type CatalogServiceConfig struct {
	BaseURL string
}

// Load reads configuration from .env file and environment variables
func Load() (*Config, error) {
	// Set common defaults using shared helper
	config.SetCommonDefaults("attendance", config.AttendanceServicePort, config.AttendanceDBPort)

	// Set service-specific defaults
	viper.SetDefault("CATALOG_SERVICE_URL", "http://localhost:"+config.CatalogServicePort)

	// Setup Viper using shared helper
	if err := config.SetupViper("attendance-service"); err != nil {
		return nil, err
	}

	// Load common config using shared helper
	server, database, rabbitmq, redis, jwt := config.LoadCommonConfig()

	// Build config struct
	cfg := &Config{
		Server:   server,
		Database: database,
		RabbitMQ: rabbitmq,
		Redis:    redis,
		JWT:      jwt,
		RateLimit: config.LoadRateLimitConfig(),
		CatalogService: CatalogServiceConfig{
			BaseURL: viper.GetString("CATALOG_SERVICE_URL"),
		},
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate common config using shared helper
	return config.ValidateCommonConfig(c.Server, c.Database, c.RabbitMQ, c.JWT)
}
