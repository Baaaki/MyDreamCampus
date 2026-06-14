package config

import (
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/spf13/viper"
)

// Config holds all configuration for the student service
type Config struct {
	Server    config.ServerConfig
	Database  config.DatabaseConfig
	RabbitMQ  config.RabbitMQConfig
	Redis     config.RedisConfig
	JWT       config.JWTConfig
	Services  ServicesConfig
	RateLimit config.RateLimitConfig
}

// ServicesConfig holds external service URLs
type ServicesConfig struct {
	StaffServiceURL string `mapstructure:"STAFF_SERVICE_URL"`
}

// Load reads configuration from .env file and environment variables
func Load() (*Config, error) {
	// Set common defaults using shared helper
	config.SetCommonDefaults("student", config.StudentServicePort, config.StudentDBPort)

	// Set student-specific defaults
	viper.SetDefault("STAFF_SERVICE_URL", "http://localhost:"+config.StaffServicePort)

	// Setup Viper using shared helper
	if err := config.SetupViper("student-service"); err != nil {
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
		Services: ServicesConfig{
			StaffServiceURL: viper.GetString("STAFF_SERVICE_URL"),
		},
		RateLimit: config.LoadRateLimitConfig(),
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
	if err := config.ValidateCommonConfig(c.Server, c.Database, c.RabbitMQ, c.JWT); err != nil {
		return err
	}

	// Validate student-specific config
	if c.Services.StaffServiceURL == "" {
		return fmt.Errorf("STAFF_SERVICE_URL is required")
	}

	return nil
}
