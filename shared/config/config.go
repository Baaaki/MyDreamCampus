package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// Config represents application configuration
type Config struct {
	Environment string        `mapstructure:"ENVIRONMENT"`
	Port        string        `mapstructure:"PORT"`
	Database    DatabaseConfig
	RabbitMQ    RabbitMQConfig
	Redis       RedisConfig
	JWT         JWTConfig
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

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig(path string) (Config, error) {
	var config Config

	viper.AddConfigPath(path)
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	// Read config file (optional - env vars take precedence)
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: .env file not found, reading from environment variables")
	}

	// Unmarshal into config struct
	if err := viper.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if err := validateConfig(&config); err != nil {
		return config, err
	}

	return config, nil
}

// validateConfig validates required configuration fields
func validateConfig(config *Config) error {
	if config.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	if config.Database.URL == "" {
		return fmt.Errorf("DB_URL is required")
	}
	if config.RabbitMQ.URL == "" {
		return fmt.Errorf("RABBITMQ_URL is required")
	}
	return nil
}

// MustLoadConfig loads config or panics
func MustLoadConfig(path string) Config {
	config, err := LoadConfig(path)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return config
}