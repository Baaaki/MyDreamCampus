package config

import (
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/spf13/viper"
)

// Config holds all configuration for the auth service
type Config struct {
	Server    config.ServerConfig
	Database  config.DatabaseConfig
	RabbitMQ  config.RabbitMQConfig
	Redis     config.RedisConfig
	JWT       JWTConfig
	Admin     AdminConfig
	Timeout   config.TimeoutConfig
	RateLimit config.RateLimitConfig
}

// JWTConfig holds JWT-related configuration (auth-specific with expiry times)
type JWTConfig struct {
	Secret             string `mapstructure:"JWT_SECRET"`
	AccessTokenExpiry  int    `mapstructure:"JWT_ACCESS_TOKEN_EXPIRY"`  // minutes
	RefreshTokenExpiry int    `mapstructure:"JWT_REFRESH_TOKEN_EXPIRY"` // hours
}

// AdminConfig holds admin user configuration
type AdminConfig struct {
	Email           string `mapstructure:"ADMIN_EMAIL"`
	InitialPassword string `mapstructure:"ADMIN_INITIAL_PASSWORD"`
}

// Load reads configuration from .env file and environment variables
func Load() (*Config, error) {
	// Set common defaults using shared helper
	config.SetCommonDefaults("auth", config.AuthServicePort, config.AuthDBPort)

	// Set auth-specific defaults
	viper.SetDefault("JWT_ACCESS_TOKEN_EXPIRY", 600) // minutes
	viper.SetDefault("JWT_REFRESH_TOKEN_EXPIRY", 24) // hours
	viper.SetDefault("ADMIN_EMAIL", "admin@university.edu.tr")
	viper.SetDefault("ADMIN_INITIAL_PASSWORD", "Admin123!")

	// Setup Viper using shared helper
	if err := config.SetupViper("auth-service"); err != nil {
		return nil, err
	}

	// Load common config using shared helper
	server, database, rabbitmq, redis, jwtBase := config.LoadCommonConfig()

	// Build config struct
	cfg := &Config{
		Server:   server,
		Database: database,
		RabbitMQ: rabbitmq,
		Redis:    redis,
		JWT: JWTConfig{
			Secret:             jwtBase.Secret,
			AccessTokenExpiry:  viper.GetInt("JWT_ACCESS_TOKEN_EXPIRY"),
			RefreshTokenExpiry: viper.GetInt("JWT_REFRESH_TOKEN_EXPIRY"),
		},
		Admin: AdminConfig{
			Email:           viper.GetString("ADMIN_EMAIL"),
			InitialPassword: viper.GetString("ADMIN_INITIAL_PASSWORD"),
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
	jwtBase := config.JWTConfig{Secret: c.JWT.Secret}
	if err := config.ValidateCommonConfig(c.Server, c.Database, c.RabbitMQ, jwtBase); err != nil {
		return err
	}

	// Validate auth-specific config
	if err := config.ValidateRedisConfig(c.Server, c.Redis); err != nil {
		return err
	}
	if c.Admin.Email == "" {
		return fmt.Errorf("ADMIN_EMAIL is required")
	}
	if c.Admin.InitialPassword == "" {
		return fmt.Errorf("ADMIN_INITIAL_PASSWORD is required")
	}
	if c.Server.Environment == "production" && c.Admin.InitialPassword == "Admin123!" {
		return fmt.Errorf("ADMIN_INITIAL_PASSWORD must be changed in production")
	}

	return nil
}
