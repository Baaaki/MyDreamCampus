package config

import (
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/spf13/viper"
)

// Config holds all configuration for the course catalog service
type Config struct {
	Server                config.ServerConfig
	Database              config.DatabaseConfig
	RabbitMQ              config.RabbitMQConfig
	Redis                 config.RedisConfig
	JWT                   config.JWTConfig
	StaffService          StaffServiceConfig
	RateLimit             config.RateLimitConfig
	InternalServiceSecret string
	EnrollmentService     ServiceURLConfig
	GradesService         ServiceURLConfig
	AttendanceService     ServiceURLConfig
	MealService           ServiceURLConfig
}

// StaffServiceConfig holds configuration for Staff Service integration
type StaffServiceConfig struct {
	BaseURL string
}

// ServiceURLConfig holds base URL for a downstream service
type ServiceURLConfig struct {
	BaseURL string
}

// Load reads configuration from .env file and environment variables
func Load() (*Config, error) {
	// Set common defaults using shared helper
	config.SetCommonDefaults("catalog", config.CatalogServicePort, config.CatalogDBPort)

	// Set service-specific defaults
	viper.SetDefault("STAFF_SERVICE_URL", "http://localhost:"+config.StaffServicePort)
	viper.SetDefault("ENROLLMENT_SERVICE_URL", "http://localhost:"+config.EnrollmentServicePort)
	viper.SetDefault("GRADES_SERVICE_URL", "http://localhost:"+config.GradesServicePort)
	viper.SetDefault("ATTENDANCE_SERVICE_URL", "http://localhost:"+config.AttendanceServicePort)
	viper.SetDefault("MEAL_SERVICE_URL", "http://localhost:"+config.MealServicePort)

	// Setup Viper using shared helper
	if err := config.SetupViper("course-catalog-service"); err != nil {
		return nil, err
	}

	// Load common config using shared helper
	server, database, rabbitmq, redis, jwt := config.LoadCommonConfig()

	// Load service-specific config
	staffService := StaffServiceConfig{
		BaseURL: viper.GetString("STAFF_SERVICE_URL"),
	}

	// Build config struct
	cfg := &Config{
		Server:                server,
		Database:              database,
		RabbitMQ:              rabbitmq,
		Redis:                 redis,
		JWT:                   jwt,
		StaffService:          staffService,
		RateLimit:             config.LoadRateLimitConfig(),
		InternalServiceSecret: viper.GetString("INTERNAL_SERVICE_SECRET"),
		EnrollmentService:     ServiceURLConfig{BaseURL: viper.GetString("ENROLLMENT_SERVICE_URL")},
		GradesService:        ServiceURLConfig{BaseURL: viper.GetString("GRADES_SERVICE_URL")},
		AttendanceService:    ServiceURLConfig{BaseURL: viper.GetString("ATTENDANCE_SERVICE_URL")},
		MealService:          ServiceURLConfig{BaseURL: viper.GetString("MEAL_SERVICE_URL")},
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

	// Validate service-specific config
	if c.StaffService.BaseURL == "" {
		return fmt.Errorf("STAFF_SERVICE_URL is required")
	}

	return nil
}
