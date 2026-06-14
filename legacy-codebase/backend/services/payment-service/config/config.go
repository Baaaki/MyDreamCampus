package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration for the payment service
type Config struct {
	Server ServerConfig
}

// ServerConfig holds server settings
type ServerConfig struct {
	Port        string `mapstructure:"PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Set defaults
	viper.SetDefault("PORT", "50051")
	viper.SetDefault("ENVIRONMENT", "development")

	// Read from .env file
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")

	// Ignore error if .env file doesn't exist
	_ = viper.ReadInConfig()

	// Read from environment variables
	viper.AutomaticEnv()

	cfg := &Config{
		Server: ServerConfig{
			Port:        viper.GetString("PORT"),
			Environment: viper.GetString("ENVIRONMENT"),
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
	if c.Server.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	return nil
}
