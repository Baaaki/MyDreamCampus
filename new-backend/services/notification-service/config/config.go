package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AppURL      string
	DatabaseURL string
	RabbitMQURL string
	SMTP        SMTPConfig
	LogLevel    string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func Load() (*Config, error) {
	viper.SetEnvPrefix("NOTIF")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("APP_URL", "http://localhost:3000")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/notification?sslmode=disable")
	viper.SetDefault("RABBITMQ_URL", "amqp://rabbitmq:rabbitmq@localhost:5672/")
	viper.SetDefault("SMTP_HOST", "localhost")
	viper.SetDefault("SMTP_PORT", 1025)
	viper.SetDefault("SMTP_USERNAME", "")
	viper.SetDefault("SMTP_PASSWORD", "")
	viper.SetDefault("SMTP_FROM", "noreply@mydreamcampus.com")
	viper.SetDefault("LOG_LEVEL", "info")

	cfg := &Config{
		AppURL:      viper.GetString("APP_URL"),
		DatabaseURL: viper.GetString("DATABASE_URL"),
		RabbitMQURL: viper.GetString("RABBITMQ_URL"),
		LogLevel:    viper.GetString("LOG_LEVEL"),
		SMTP: SMTPConfig{
			Host:     viper.GetString("SMTP_HOST"),
			Port:     viper.GetInt("SMTP_PORT"),
			Username: viper.GetString("SMTP_USERNAME"),
			Password: viper.GetString("SMTP_PASSWORD"),
			From:     viper.GetString("SMTP_FROM"),
		},
	}

	return cfg, nil
}
