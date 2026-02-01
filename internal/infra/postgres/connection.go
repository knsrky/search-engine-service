// Package postgres provides PostgreSQL database connection and repository implementation.
package postgres

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Config holds database connection configuration.
type Config struct {
	Host         string
	Port         int
	Name         string
	User         string
	Password     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// NewConnection creates a new GORM database connection.
func NewConnection(cfg Config, logger *zap.Logger) (*gorm.DB, error) {
	// Configure GORM logger
	gormLogLevel := gormlogger.Silent
	if logger != nil {
		gormLogLevel = gormlogger.Info
	}

	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormLogLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: true, // Cache prepared statements
	}

	// Open connection
	db, err := gorm.Open(postgres.Open(cfg.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("getting underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	if logger != nil {
		logger.Info("database connection established",
			zap.String("host", cfg.Host),
			zap.Int("port", cfg.Port),
			zap.String("database", cfg.Name),
		)
	}

	return db, nil
}

// Close closes the database connection.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// HealthCheck verifies the database connection is alive.
func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}
