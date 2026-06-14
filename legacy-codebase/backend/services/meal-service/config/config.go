package config

import (
	"fmt"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/spf13/viper"
)

// Config holds all configuration for the meal service
type Config struct {
	Server         config.ServerConfig
	Database       config.DatabaseConfig
	RabbitMQ       config.RabbitMQConfig
	Redis          config.RedisConfig
	JWT            config.JWTConfig
	RateLimit      config.RateLimitConfig
	Payment        PaymentConfig
	QR             QRConfig
	Reservation    ReservationConfig
	MealTime       MealTimeConfig
	Outbox         OutboxConfig
	CatalogService CatalogServiceConfig
}

// CatalogServiceConfig holds configuration for Catalog Service integration
type CatalogServiceConfig struct {
	BaseURL string
}

// PaymentConfig holds payment service configuration (gRPC)
type PaymentConfig struct {
	GRPCAddress string `mapstructure:"PAYMENT_GRPC_ADDRESS"`
}

// QRConfig holds QR code configuration
type QRConfig struct {
	Secret string `mapstructure:"QR_SECRET"`
}

// ReservationConfig holds reservation settings
type ReservationConfig struct {
	TimeoutMinutes           int     `mapstructure:"RESERVATION_TIMEOUT_MINUTES"`
	MealPriceTRY             float64 `mapstructure:"MEAL_PRICE_TRY"`
	CancelCutoffHours        int     `mapstructure:"RESERVATION_CANCEL_CUTOFF_HOURS"`
	QRValidityWindowSeconds  int     `mapstructure:"QR_VALIDITY_WINDOW_SECONDS"`
}

// MealTimeConfig holds meal time window configuration (UTC+3)
type MealTimeConfig struct {
	LunchStartHour  int `mapstructure:"LUNCH_START_HOUR"`
	LunchEndHour    int `mapstructure:"LUNCH_END_HOUR"`
	DinnerStartHour int `mapstructure:"DINNER_START_HOUR"`
	DinnerEndHour   int `mapstructure:"DINNER_END_HOUR"`
}

// OutboxConfig holds outbox pattern configuration
type OutboxConfig struct {
	PollIntervalSeconds int `mapstructure:"OUTBOX_POLL_INTERVAL_SECONDS"`
	BatchSize           int `mapstructure:"OUTBOX_BATCH_SIZE"`
	MaxRetries          int `mapstructure:"OUTBOX_MAX_RETRIES"`
}

// Load reads configuration from .env file and environment variables
func Load() (*Config, error) {
	// Set common defaults using shared helper
	config.SetCommonDefaults("meal", config.MealServicePort, config.MealDBPort)

	// Set service-specific defaults
	viper.SetDefault("CATALOG_SERVICE_URL", "http://localhost:"+config.CatalogServicePort)

	// Set meal-specific defaults
	setMealDefaults()

	// Setup Viper using shared helper
	if err := config.SetupViper("meal-service"); err != nil {
		return nil, err
	}

	// Load common config using shared helper
	server, database, rabbitmq, redis, jwt := config.LoadCommonConfig()

	// Load meal-specific config
	payment := PaymentConfig{
		GRPCAddress: viper.GetString("PAYMENT_GRPC_ADDRESS"),
	}

	qr := QRConfig{
		Secret: viper.GetString("QR_SECRET"),
	}

	reservation := ReservationConfig{
		TimeoutMinutes:          viper.GetInt("RESERVATION_TIMEOUT_MINUTES"),
		MealPriceTRY:            viper.GetFloat64("MEAL_PRICE_TRY"),
		CancelCutoffHours:       viper.GetInt("RESERVATION_CANCEL_CUTOFF_HOURS"),
		QRValidityWindowSeconds: viper.GetInt("QR_VALIDITY_WINDOW_SECONDS"),
	}

	mealTime := MealTimeConfig{
		LunchStartHour:  viper.GetInt("LUNCH_START_HOUR"),
		LunchEndHour:    viper.GetInt("LUNCH_END_HOUR"),
		DinnerStartHour: viper.GetInt("DINNER_START_HOUR"),
		DinnerEndHour:   viper.GetInt("DINNER_END_HOUR"),
	}

	outbox := OutboxConfig{
		PollIntervalSeconds: viper.GetInt("OUTBOX_POLL_INTERVAL_SECONDS"),
		BatchSize:           viper.GetInt("OUTBOX_BATCH_SIZE"),
		MaxRetries:          viper.GetInt("OUTBOX_MAX_RETRIES"),
	}

	// Build config struct
	cfg := &Config{
		Server:      server,
		Database:    database,
		RabbitMQ:    rabbitmq,
		Redis:       redis,
		JWT:         jwt,
		RateLimit:   config.LoadRateLimitConfig(),
		Payment:     payment,
		QR:          qr,
		Reservation: reservation,
		MealTime:    mealTime,
		Outbox:      outbox,
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

// setMealDefaults sets meal-specific default configuration values
func setMealDefaults() {
	// Payment service gRPC address
	viper.SetDefault("PAYMENT_GRPC_ADDRESS", "localhost:50051")

	// QR Code
	viper.SetDefault("QR_SECRET", "change-this-qr-secret-in-production")

	// Reservation settings
	viper.SetDefault("RESERVATION_TIMEOUT_MINUTES", 15)
	viper.SetDefault("MEAL_PRICE_TRY", 15.00)
	viper.SetDefault("RESERVATION_CANCEL_CUTOFF_HOURS", 2)
	viper.SetDefault("QR_VALIDITY_WINDOW_SECONDS", 30)

	// Meal time windows (UTC+3)
	viper.SetDefault("LUNCH_START_HOUR", 11)
	viper.SetDefault("LUNCH_END_HOUR", 13)
	viper.SetDefault("DINNER_START_HOUR", 16)
	viper.SetDefault("DINNER_END_HOUR", 19)

	// Outbox settings
	viper.SetDefault("OUTBOX_POLL_INTERVAL_SECONDS", 5)
	viper.SetDefault("OUTBOX_BATCH_SIZE", 50)
	viper.SetDefault("OUTBOX_MAX_RETRIES", 5)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate common config using shared helper
	if err := config.ValidateCommonConfig(c.Server, c.Database, c.RabbitMQ, c.JWT); err != nil {
		return err
	}

	// Validate meal-specific config
	if c.Payment.GRPCAddress == "" {
		return fmt.Errorf("PAYMENT_GRPC_ADDRESS is required")
	}

	if c.QR.Secret == "" {
		return fmt.Errorf("QR_SECRET is required")
	}

	if c.QR.Secret == "change-this-qr-secret-in-production" && c.Server.Environment == "production" {
		return fmt.Errorf("QR_SECRET must be changed in production")
	}

	if c.Reservation.TimeoutMinutes <= 0 {
		return fmt.Errorf("RESERVATION_TIMEOUT_MINUTES must be positive")
	}

	if c.Reservation.MealPriceTRY <= 0 {
		return fmt.Errorf("MEAL_PRICE_TRY must be positive")
	}

	if c.Reservation.CancelCutoffHours < 0 {
		return fmt.Errorf("RESERVATION_CANCEL_CUTOFF_HOURS must be non-negative")
	}

	if c.Reservation.QRValidityWindowSeconds <= 0 {
		return fmt.Errorf("QR_VALIDITY_WINDOW_SECONDS must be positive")
	}

	// Validate meal time windows
	if c.MealTime.LunchStartHour < 0 || c.MealTime.LunchStartHour > 23 {
		return fmt.Errorf("LUNCH_START_HOUR must be between 0 and 23")
	}
	if c.MealTime.LunchEndHour < 0 || c.MealTime.LunchEndHour > 23 {
		return fmt.Errorf("LUNCH_END_HOUR must be between 0 and 23")
	}
	if c.MealTime.DinnerStartHour < 0 || c.MealTime.DinnerStartHour > 23 {
		return fmt.Errorf("DINNER_START_HOUR must be between 0 and 23")
	}
	if c.MealTime.DinnerEndHour < 0 || c.MealTime.DinnerEndHour > 23 {
		return fmt.Errorf("DINNER_END_HOUR must be between 0 and 23")
	}

	// Validate outbox config
	if c.Outbox.PollIntervalSeconds <= 0 {
		return fmt.Errorf("OUTBOX_POLL_INTERVAL_SECONDS must be positive")
	}
	if c.Outbox.BatchSize <= 0 {
		return fmt.Errorf("OUTBOX_BATCH_SIZE must be positive")
	}
	if c.Outbox.MaxRetries < 0 {
		return fmt.Errorf("OUTBOX_MAX_RETRIES must be non-negative")
	}

	return nil
}
