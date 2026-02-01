// Package config provides application configuration management using Viper.
// Configuration is loaded from YAML files and environment variables.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Provider ProviderConfig `mapstructure:"provider"`
	Sync     SyncConfig     `mapstructure:"sync"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Sentry   SentryConfig   `mapstructure:"sentry"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Cache    CacheConfig    `mapstructure:"cache"`
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Name  string `mapstructure:"name"`
	Env   string `mapstructure:"env"` // development, staging, production
	Port  int    `mapstructure:"port"`
	Debug bool   `mapstructure:"debug"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Name         string        `mapstructure:"name"`
	User         string        `mapstructure:"user"`
	Password     string        `mapstructure:"password"`
	SSLMode      string        `mapstructure:"ssl_mode"`
	MaxOpenConns int           `mapstructure:"max_open_conns"`
	MaxIdleConns int           `mapstructure:"max_idle_conns"`
	MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// ProviderConfig holds external provider settings.
type ProviderConfig struct {
	A ProviderEndpoint `mapstructure:"a"`
	B ProviderEndpoint `mapstructure:"b"`
}

// ProviderEndpoint holds a single provider's configuration.
type ProviderEndpoint struct {
	BaseURL string        `mapstructure:"base_url"`
	Timeout time.Duration `mapstructure:"timeout"`
	Retry   RetryConfig   `mapstructure:"retry"`
	CB      CBConfig      `mapstructure:"circuit_breaker"`
}

// RetryConfig holds retry settings.
type RetryConfig struct {
	MaxAttempts int           `mapstructure:"max_attempts"`
	WaitTime    time.Duration `mapstructure:"wait_time"`
	MaxWaitTime time.Duration `mapstructure:"max_wait_time"`
}

// CBConfig holds circuit breaker settings.
type CBConfig struct {
	MaxRequests  uint32        `mapstructure:"max_requests"`
	Interval     time.Duration `mapstructure:"interval"`
	Timeout      time.Duration `mapstructure:"timeout"`
	FailureRatio float64       `mapstructure:"failure_ratio"`
}

// SyncConfig holds background sync worker settings.
type SyncConfig struct {
	Interval  time.Duration `mapstructure:"interval"`
	OnStartup bool          `mapstructure:"on_startup"`
	Timeout   time.Duration `mapstructure:"timeout"`
	BatchSize int           `mapstructure:"batch_size"`
}

// LoggerConfig holds logging settings.
type LoggerConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, console
	Output string `mapstructure:"output"` // stdout, stderr, file path
}

// SentryConfig holds Sentry error tracking settings.
type SentryConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	DSN         string  `mapstructure:"dsn"`
	Environment string  `mapstructure:"environment"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

// RedisConfig holds Redis connection settings for distributed locking.
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// CacheConfig holds caching settings.
type CacheConfig struct {
	Enabled   bool          `mapstructure:"enabled"`
	SearchTTL time.Duration `mapstructure:"search_ttl"`
	KeyPrefix string        `mapstructure:"key_prefix"`
}

// Load reads configuration from file and environment variables.
// Priority: env vars > config file > defaults
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Config file settings
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		// Config file not found, continue with defaults + env vars
	}

	// Environment variable settings
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("app.name", "search-engine-service")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.debug", true)

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "search_engine")
	v.SetDefault("database.user", "app")
	v.SetDefault("database.password", "secret")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.max_lifetime", "5m")

	// Provider A defaults
	v.SetDefault("provider.a.base_url", "http://localhost:8081")
	v.SetDefault("provider.a.timeout", "10s")
	v.SetDefault("provider.a.retry.max_attempts", 3)
	v.SetDefault("provider.a.retry.wait_time", "1s")
	v.SetDefault("provider.a.retry.max_wait_time", "5s")
	v.SetDefault("provider.a.circuit_breaker.max_requests", 3)
	v.SetDefault("provider.a.circuit_breaker.interval", "60s")
	v.SetDefault("provider.a.circuit_breaker.timeout", "30s")
	v.SetDefault("provider.a.circuit_breaker.failure_ratio", 0.5)

	// Provider B defaults
	v.SetDefault("provider.b.base_url", "http://localhost:8082")
	v.SetDefault("provider.b.timeout", "10s")
	v.SetDefault("provider.b.retry.max_attempts", 3)
	v.SetDefault("provider.b.retry.wait_time", "1s")
	v.SetDefault("provider.b.retry.max_wait_time", "5s")
	v.SetDefault("provider.b.circuit_breaker.max_requests", 3)
	v.SetDefault("provider.b.circuit_breaker.interval", "60s")
	v.SetDefault("provider.b.circuit_breaker.timeout", "30s")
	v.SetDefault("provider.b.circuit_breaker.failure_ratio", 0.5)

	// Sync defaults
	v.SetDefault("sync.interval", "5m")
	v.SetDefault("sync.on_startup", true)
	v.SetDefault("sync.timeout", "30s")
	v.SetDefault("sync.batch_size", 100)

	// Logger defaults
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "console")
	v.SetDefault("logger.output", "stdout")

	// Sentry defaults
	v.SetDefault("sentry.enabled", false)
	v.SetDefault("sentry.dsn", "")
	v.SetDefault("sentry.environment", "development")
	v.SetDefault("sentry.sample_rate", 1.0)

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// Cache defaults
	v.SetDefault("cache.enabled", false)
	v.SetDefault("cache.search_ttl", "15m")
	v.SetDefault("cache.key_prefix", "search-engine")
}
