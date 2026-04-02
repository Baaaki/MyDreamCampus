package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration for the auth service
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Admin    AdminConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Environment string `mapstructure:"ENVIRONMENT"`
	Port        string `mapstructure:"PORT"`
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	URL string `mapstructure:"DB_URL"`
}

// RabbitMQConfig holds RabbitMQ-related configuration
type RabbitMQConfig struct {
	URL string `mapstructure:"RABBITMQ_URL"`
}

// RedisConfig holds Redis-related configuration
type RedisConfig struct {
	Addr     string `mapstructure:"REDIS_ADDR"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	Secret               string `mapstructure:"JWT_SECRET"`
	AccessTokenExpiry    int    `mapstructure:"JWT_ACCESS_TOKEN_EXPIRY"`  // minutes
	RefreshTokenExpiry   int    `mapstructure:"JWT_REFRESH_TOKEN_EXPIRY"` // hours
}

// AdminConfig holds admin user configuration
type AdminConfig struct {
	Email           string `mapstructure:"ADMIN_EMAIL"`
	InitialPassword string `mapstructure:"ADMIN_INITIAL_PASSWORD"`
}

// Load reads configuration from .env file and environment variables
func Load() (*Config, error) {
	// Set config file type
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	// Read from environment variables (overrides .env)
	viper.AutomaticEnv()

	// Read config file (optional, env vars take priority)
	_ = viper.ReadInConfig()

	// Set default values
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("JWT_ACCESS_TOKEN_EXPIRY", 15)
	viper.SetDefault("JWT_REFRESH_TOKEN_EXPIRY", 24)
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("ADMIN_EMAIL", "admin@university.edu.tr")
	viper.SetDefault("ADMIN_INITIAL_PASSWORD", "Admin123!")

	// Build config from env vars
	config := &Config{
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
			Secret:             viper.GetString("JWT_SECRET"),
			AccessTokenExpiry:  viper.GetInt("JWT_ACCESS_TOKEN_EXPIRY"),
			RefreshTokenExpiry: viper.GetInt("JWT_REFRESH_TOKEN_EXPIRY"),
		},
		Admin: AdminConfig{
			Email:           viper.GetString("ADMIN_EMAIL"),
			InitialPassword: viper.GetString("ADMIN_INITIAL_PASSWORD"),
		},
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
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
	if c.Redis.Addr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}
	if c.Admin.Email == "" {
		return fmt.Errorf("ADMIN_EMAIL is required")
	}
	if c.Admin.InitialPassword == "" {
		return fmt.Errorf("ADMIN_INITIAL_PASSWORD is required")
	}
	return nil
}
