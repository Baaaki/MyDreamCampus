package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

// Config holds one service's configuration. Every service loads the same
// struct from its own environment; there is no per-service config type.
type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	RabbitMQ       RabbitMQConfig
	Redis          RedisConfig
	JWT            JWTConfig
	Admin          AdminConfig
	Outbox         OutboxConfig
	QR             QRConfig
	Reservation    ReservationConfig
	MealTime       MealTimeConfig
	RateLimit      RateLimitConfig
	Timeout        TimeoutConfig
	InternalClient InternalClientConfig
}

type ServerConfig struct {
	Environment string `mapstructure:"ENVIRONMENT"`
	Port        string `mapstructure:"PORT"`
	// InternalSecret authenticates loopback service-to-service calls
	// (X-Internal-Secret header on /internal/* routes).
	InternalSecret string `mapstructure:"INTERNAL_SERVICE_SECRET"`
}

// InternalClientConfig governs how services reach each other: internal REST
// with a shared secret, one entry per target.
type InternalClientConfig struct {
	// ServiceURLs is keyed by target name (staff, student, catalog,
	// payment, meal).
	ServiceURLs    map[string]string
	TimeoutSeconds int `mapstructure:"INTERNAL_CLIENT_TIMEOUT_SECONDS"`
	Breaker        CircuitBreakerConfig
}

// CircuitBreakerConfig tunes the per-target breaker in shared/client.
type CircuitBreakerConfig struct {
	MaxRequests         int `mapstructure:"CIRCUIT_BREAKER_MAX_REQUESTS"`
	TimeoutSeconds      int `mapstructure:"CIRCUIT_BREAKER_TIMEOUT_SECONDS"`
	ConsecutiveFailures int `mapstructure:"CIRCUIT_BREAKER_CONSECUTIVE_FAILURES"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"DB_URL"`
}

type RabbitMQConfig struct {
	URL string `mapstructure:"RABBITMQ_URL"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"REDIS_ADDR"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

type JWTConfig struct {
	Secret             string `mapstructure:"JWT_SECRET"`
	AccessTokenExpiry  int    `mapstructure:"JWT_ACCESS_TOKEN_EXPIRY"`
	RefreshTokenExpiry int    `mapstructure:"JWT_REFRESH_TOKEN_EXPIRY"`
}

type AdminConfig struct {
	Email           string `mapstructure:"ADMIN_EMAIL"`
	InitialPassword string `mapstructure:"ADMIN_INITIAL_PASSWORD"`
}

// OutboxConfig governs the per-module outbox relay goroutines
// (one worker per module).
type OutboxConfig struct {
	IntervalSeconds int `mapstructure:"OUTBOX_POLL_INTERVAL_SECONDS"`
	BatchSize       int `mapstructure:"OUTBOX_BATCH_SIZE"`
	MaxRetries      int `mapstructure:"OUTBOX_MAX_RETRIES"`
}

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

type TimeoutConfig struct {
	RequestTimeoutSeconds         int `mapstructure:"REQUEST_TIMEOUT_SECONDS"`
	ShutdownTimeoutSeconds        int `mapstructure:"SHUTDOWN_TIMEOUT_SECONDS"`
	AccountLockDurationMinutes    int `mapstructure:"ACCOUNT_LOCK_DURATION_MINUTES"`
	CleanupSchedulerIntervalHours int `mapstructure:"CLEANUP_SCHEDULER_INTERVAL_HOURS"`
	ProcessedEventsRetentionDays  int `mapstructure:"PROCESSED_EVENTS_RETENTION_DAYS"`
}

type QRConfig struct {
	Secret string `mapstructure:"QR_SECRET"`
}

type ReservationConfig struct {
	TimeoutMinutes          int     `mapstructure:"RESERVATION_TIMEOUT_MINUTES"`
	MealPriceTRY            float64 `mapstructure:"MEAL_PRICE_TRY"`
	CancelCutoffHours       int     `mapstructure:"RESERVATION_CANCEL_CUTOFF_HOURS"`
	QRValidityWindowSeconds int     `mapstructure:"QR_VALIDITY_WINDOW_SECONDS"`
}

type MealTimeConfig struct {
	LunchStartHour  int `mapstructure:"LUNCH_START_HOUR"`
	LunchEndHour    int `mapstructure:"LUNCH_END_HOUR"`
	DinnerStartHour int `mapstructure:"DINNER_START_HOUR"`
	DinnerEndHour   int `mapstructure:"DINNER_END_HOUR"`
}

const minJWTSecretLength = 32

// Load reads .env + environment, applies defaults, validates and returns Config.
func Load() (*Config, error) {
	setDefaults()

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath("/etc/mydreamcampus/")
	viper.AddConfigPath("$HOME/.mydreamcampus")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Environment:    viper.GetString("ENVIRONMENT"),
			Port:           viper.GetString("PORT"),
			InternalSecret: viper.GetString("INTERNAL_SERVICE_SECRET"),
		},
		Database: DatabaseConfig{URL: viper.GetString("DB_URL")},
		RabbitMQ: RabbitMQConfig{URL: viper.GetString("RABBITMQ_URL")},
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
		Outbox: OutboxConfig{
			IntervalSeconds: viper.GetInt("OUTBOX_POLL_INTERVAL_SECONDS"),
			BatchSize:       viper.GetInt("OUTBOX_BATCH_SIZE"),
			MaxRetries:      viper.GetInt("OUTBOX_MAX_RETRIES"),
		},
		RateLimit: RateLimitConfig{
			Enabled:            viper.GetBool("RATE_LIMIT_ENABLED"),
			IPLimit:            viper.GetInt("RATE_LIMIT_IP_LIMIT"),
			IPWindowSecs:       viper.GetInt("RATE_LIMIT_IP_WINDOW_SECS"),
			UserLimit:          viper.GetInt("RATE_LIMIT_USER_LIMIT"),
			UserWindowSecs:     viper.GetInt("RATE_LIMIT_USER_WINDOW_SECS"),
			LoginLimit:         viper.GetInt("RATE_LIMIT_LOGIN_LIMIT"),
			LoginWindowSecs:    viper.GetInt("RATE_LIMIT_LOGIN_WINDOW_SECS"),
			RefreshLimit:       viper.GetInt("RATE_LIMIT_REFRESH_LIMIT"),
			RefreshWindowSecs:  viper.GetInt("RATE_LIMIT_REFRESH_WINDOW_SECS"),
			PasswordLimit:      viper.GetInt("RATE_LIMIT_PASSWORD_LIMIT"),
			PasswordWindowSecs: viper.GetInt("RATE_LIMIT_PASSWORD_WINDOW_SECS"),
		},
		Timeout: TimeoutConfig{
			RequestTimeoutSeconds:         viper.GetInt("REQUEST_TIMEOUT_SECONDS"),
			ShutdownTimeoutSeconds:        viper.GetInt("SHUTDOWN_TIMEOUT_SECONDS"),
			AccountLockDurationMinutes:    viper.GetInt("ACCOUNT_LOCK_DURATION_MINUTES"),
			CleanupSchedulerIntervalHours: viper.GetInt("CLEANUP_SCHEDULER_INTERVAL_HOURS"),
			ProcessedEventsRetentionDays:  viper.GetInt("PROCESSED_EVENTS_RETENTION_DAYS"),
		},
		// Every other section is read here explicitly, so a section left out
		// silently stays at its zero value however good its SetDefault is.
		// This one was: meal priced reservations at 0 TRY and gave QR codes a
		// 0-second validity window.
		Reservation: ReservationConfig{
			TimeoutMinutes:          viper.GetInt("RESERVATION_TIMEOUT_MINUTES"),
			MealPriceTRY:            viper.GetFloat64("MEAL_PRICE_TRY"),
			CancelCutoffHours:       viper.GetInt("RESERVATION_CANCEL_CUTOFF_HOURS"),
			QRValidityWindowSeconds: viper.GetInt("QR_VALIDITY_WINDOW_SECONDS"),
		},
		InternalClient: InternalClientConfig{
			ServiceURLs: map[string]string{
				"staff":   viper.GetString("STAFF_SERVICE_URL"),
				"student": viper.GetString("STUDENT_SERVICE_URL"),
				"catalog": viper.GetString("CATALOG_SERVICE_URL"),
				"payment": viper.GetString("PAYMENT_SERVICE_URL"),
				"meal":    viper.GetString("MEAL_SERVICE_URL"),
			},
			TimeoutSeconds: viper.GetInt("INTERNAL_CLIENT_TIMEOUT_SECONDS"),
			Breaker: CircuitBreakerConfig{
				MaxRequests:         viper.GetInt("CIRCUIT_BREAKER_MAX_REQUESTS"),
				TimeoutSeconds:      viper.GetInt("CIRCUIT_BREAKER_TIMEOUT_SECONDS"),
				ConsecutiveFailures: viper.GetInt("CIRCUIT_BREAKER_CONSECUTIVE_FAILURES"),
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}
	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("INTERNAL_SERVICE_SECRET", "change-this-internal-secret-in-production")

	// No DB_URL default: every service owns a different database, so any value
	// here would be wrong for eight of the nine. Services that need a pool
	// fail loudly in bootstrap when it is unset.
	viper.SetDefault("RABBITMQ_URL", "amqp://rabbitmq:rabbitmq@localhost:5672/")
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "changeme_redis_secret")
	viper.SetDefault("REDIS_DB", 0)

	viper.SetDefault("JWT_SECRET", "change-this-secret-in-production")
	// Access expiry is in MINUTES (see generateAccessToken), refresh in HOURS.
	// Keep access short: revocation relies on Redis blacklist + short TTL.
	viper.SetDefault("JWT_ACCESS_TOKEN_EXPIRY", 15)
	viper.SetDefault("JWT_REFRESH_TOKEN_EXPIRY", 24)

	viper.SetDefault("ADMIN_EMAIL", "admin@university.edu.tr")
	viper.SetDefault("ADMIN_INITIAL_PASSWORD", "Admin123!")

	viper.SetDefault("OUTBOX_POLL_INTERVAL_SECONDS", 5)
	viper.SetDefault("OUTBOX_BATCH_SIZE", 10)
	viper.SetDefault("OUTBOX_MAX_RETRIES", 5)

	viper.SetDefault("RATE_LIMIT_ENABLED", true)
	viper.SetDefault("RATE_LIMIT_IP_LIMIT", 100)
	viper.SetDefault("RATE_LIMIT_IP_WINDOW_SECS", 60)
	viper.SetDefault("RATE_LIMIT_USER_LIMIT", 200)
	viper.SetDefault("RATE_LIMIT_USER_WINDOW_SECS", 60)
	viper.SetDefault("RATE_LIMIT_LOGIN_LIMIT", 20)
	viper.SetDefault("RATE_LIMIT_LOGIN_WINDOW_SECS", 60)
	viper.SetDefault("RATE_LIMIT_REFRESH_LIMIT", 10)
	viper.SetDefault("RATE_LIMIT_REFRESH_WINDOW_SECS", 60)
	viper.SetDefault("RATE_LIMIT_PASSWORD_LIMIT", 3)
	viper.SetDefault("RATE_LIMIT_PASSWORD_WINDOW_SECS", 300)

	viper.SetDefault("REQUEST_TIMEOUT_SECONDS", 10)
	viper.SetDefault("SHUTDOWN_TIMEOUT_SECONDS", 30)
	viper.SetDefault("ACCOUNT_LOCK_DURATION_MINUTES", 30)
	viper.SetDefault("CLEANUP_SCHEDULER_INTERVAL_HOURS", 1)
	viper.SetDefault("PROCESSED_EVENTS_RETENTION_DAYS", 30)

	viper.SetDefault("INTERNAL_CLIENT_TIMEOUT_SECONDS", 10)
	// Five consecutive failures before tripping: one blip must not open the
	// breaker. 30s open window covers a container restart (~5-10s).
	viper.SetDefault("CIRCUIT_BREAKER_MAX_REQUESTS", 1)
	viper.SetDefault("CIRCUIT_BREAKER_TIMEOUT_SECONDS", 30)
	viper.SetDefault("CIRCUIT_BREAKER_CONSECUTIVE_FAILURES", 5)

	// Meal service defaults
	viper.SetDefault("QR_SECRET", "change-this-qr-secret-in-production")
	viper.SetDefault("RESERVATION_TIMEOUT_MINUTES", 15)
	viper.SetDefault("MEAL_PRICE_TRY", 15.00)
	viper.SetDefault("RESERVATION_CANCEL_CUTOFF_HOURS", 2)
	viper.SetDefault("QR_VALIDITY_WINDOW_SECONDS", 30)
	viper.SetDefault("LUNCH_START_HOUR", 11)
	viper.SetDefault("LUNCH_END_HOUR", 13)
	viper.SetDefault("DINNER_START_HOUR", 16)
	viper.SetDefault("DINNER_END_HOUR", 19)
}

func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	// DB_URL is not checked here — payment owns no schema and starts without
	// one. Services that need a pool assert it during bootstrap.
	if c.RabbitMQ.URL == "" {
		return fmt.Errorf("RABBITMQ_URL is required")
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.Server.Environment == "production" {
		if c.JWT.Secret == "change-this-secret-in-production" {
			return fmt.Errorf("JWT_SECRET must be changed in production")
		}
		if len(c.JWT.Secret) < minJWTSecretLength {
			return fmt.Errorf("JWT_SECRET must be at least %d bytes in production (got %d)", minJWTSecretLength, len(c.JWT.Secret))
		}
		if c.Redis.Password == "" || c.Redis.Password == "changeme_redis_secret" {
			return fmt.Errorf("REDIS_PASSWORD must be set to a real value in production")
		}
		if c.Admin.InitialPassword == "Admin123!" {
			return fmt.Errorf("ADMIN_INITIAL_PASSWORD must be changed in production")
		}
		if c.Server.InternalSecret == "change-this-internal-secret-in-production" || len(c.Server.InternalSecret) < minJWTSecretLength {
			return fmt.Errorf("INTERNAL_SERVICE_SECRET must be set to a random value of at least %d bytes in production", minJWTSecretLength)
		}
	}
	if c.Admin.Email == "" {
		return fmt.Errorf("ADMIN_EMAIL is required")
	}
	if c.Outbox.IntervalSeconds <= 0 {
		return fmt.Errorf("OUTBOX_POLL_INTERVAL_SECONDS must be positive")
	}
	if c.Outbox.BatchSize <= 0 {
		return fmt.Errorf("OUTBOX_BATCH_SIZE must be positive")
	}
	return nil
}
