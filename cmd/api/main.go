// Package main is the entry point for the search-engine-service API.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"search-engine-service/internal/app/service"
	"search-engine-service/internal/config"
	"search-engine-service/internal/domain"
	"search-engine-service/internal/infra/postgres"
	"search-engine-service/internal/infra/postgres/migrations"
	"search-engine-service/internal/infra/provider"
	"search-engine-service/internal/infra/provider/provider_a"
	"search-engine-service/internal/infra/provider/provider_b"
	rediscache "search-engine-service/internal/infra/redis"
	"search-engine-service/internal/job"
	"search-engine-service/internal/logger"
	"search-engine-service/internal/transport/httpserver"
	"search-engine-service/internal/validator"
	"search-engine-service/pkg/locker"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Initialize logger
	log, err := logger.New(
		logger.Config{
			Level:  cfg.Logger.Level,
			Format: cfg.Logger.Format,
			Output: cfg.Logger.Output,
		},
		logger.SentryConfig{
			Enabled:     cfg.Sentry.Enabled,
			DSN:         cfg.Sentry.DSN,
			Environment: cfg.Sentry.Environment,
			SampleRate:  cfg.Sentry.SampleRate,
		},
	)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer func() { _ = log.Sync() }()

	log.Info("starting search-engine-service",
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
	)

	// Connect to database
	db, err := postgres.NewConnection(
		postgres.Config{
			Host:         cfg.Database.Host,
			Port:         cfg.Database.Port,
			Name:         cfg.Database.Name,
			User:         cfg.Database.User,
			Password:     cfg.Database.Password,
			SSLMode:      cfg.Database.SSLMode,
			MaxOpenConns: cfg.Database.MaxOpenConns,
			MaxIdleConns: cfg.Database.MaxIdleConns,
			MaxLifetime:  cfg.Database.MaxLifetime,
		},
		log.Logger,
	)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer func() { _ = postgres.Close(db) }()

	// Run migrations
	if err := migrations.Run(db); err != nil {
		log.Fatal("failed to run migrations", zap.Error(err))
	}
	log.Info("database migrations completed")

	// Create repository
	repo := postgres.NewRepository(db)

	// Create provider clients
	providerA := provider_a.New(
		provider.ClientConfig{
			BaseURL:  cfg.Provider.A.BaseURL,
			Endpoint: cfg.Provider.A.Endpoint,
			Timeout:  cfg.Provider.A.Timeout,
			Retry: provider.RetryConfig{
				MaxAttempts: cfg.Provider.A.Retry.MaxAttempts,
				WaitTime:    cfg.Provider.A.Retry.WaitTime,
				MaxWaitTime: cfg.Provider.A.Retry.MaxWaitTime,
			},
			CB: provider.CBConfig{
				MaxRequests:  cfg.Provider.A.CB.MaxRequests,
				Interval:     cfg.Provider.A.CB.Interval,
				Timeout:      cfg.Provider.A.CB.Timeout,
				FailureRatio: cfg.Provider.A.CB.FailureRatio,
			},
		},
		log.Logger,
	)

	providerB := provider_b.New(
		provider.ClientConfig{
			BaseURL:  cfg.Provider.B.BaseURL,
			Endpoint: cfg.Provider.B.Endpoint,
			Timeout:  cfg.Provider.B.Timeout,
			Retry: provider.RetryConfig{
				MaxAttempts: cfg.Provider.B.Retry.MaxAttempts,
				WaitTime:    cfg.Provider.B.Retry.WaitTime,
				MaxWaitTime: cfg.Provider.B.Retry.MaxWaitTime,
			},
			CB: provider.CBConfig{
				MaxRequests:  cfg.Provider.B.CB.MaxRequests,
				Interval:     cfg.Provider.B.CB.Interval,
				Timeout:      cfg.Provider.B.CB.Timeout,
				FailureRatio: cfg.Provider.B.CB.FailureRatio,
			},
		},
		log.Logger,
	)

	// Create domain providers slice
	domainProviders := []domain.Provider{providerA, providerB}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Ping Redis to verify connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer func() { _ = redisClient.Close() }()
	log.Info("connected to Redis",
		zap.String("host", cfg.Redis.Host),
		zap.Int("port", cfg.Redis.Port),
	)

	// Create cache implementation (optional, based on config)
	var cache domain.Cache
	if cfg.Cache.Enabled {
		cache = rediscache.NewCache(redisClient, log.Logger, cfg.Cache.KeyPrefix)
		log.Info("cache enabled",
			zap.Duration("search_ttl", cfg.Cache.SearchTTL),
			zap.String("key_prefix", cfg.Cache.KeyPrefix),
		)
	} else {
		log.Info("cache disabled")
	}

	// Create services
	searchSvc := service.NewSearchService(repo, cache, cfg.Cache.SearchTTL, log.Logger)
	syncSvc := service.NewSyncService(repo, domainProviders, log.Logger)

	// Create distributed locker
	distLocker := locker.NewRedisLocker(redisClient, log.Logger)

	// Create validator
	v := validator.New()

	// Create HTTP server
	server := httpserver.NewServer(
		httpserver.ServerConfig{
			Port:      cfg.App.Port,
			BodyLimit: 1024 * 1024, // 1MB
			Debug:     cfg.App.Debug,
		},
		searchSvc,
		syncSvc,
		db,
		v,
		log.Logger,
	)

	// Start sync scheduler with distributed locking
	scheduler := job.NewSyncScheduler(
		syncSvc,
		job.SyncConfig{
			Interval:  cfg.Sync.Interval,
			Timeout:   cfg.Sync.Timeout,
			OnStartup: cfg.Sync.OnStartup,
		},
		log.Logger,
		distLocker,
	)
	scheduler.Start(cfg.Sync.OnStartup)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info("shutdown signal received")

		// Stop scheduler
		scheduler.Stop()

		// Shutdown server with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.App.ShutdownWithContext(ctx); err != nil {
			log.Error("server shutdown error", zap.Error(err))
		}
	}()

	// Start server
	if err := server.Start(cfg.App.Port); err != nil {
		log.Fatal("server error", zap.Error(err))
	}
}
