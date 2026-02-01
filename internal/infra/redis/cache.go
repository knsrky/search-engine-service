package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Cache implements the domain.Cache interface using Redis.
// It provides key-value storage with TTL support and prefix-based namespacing.
type Cache struct {
	client    *redis.Client
	logger    *zap.Logger
	keyPrefix string
}

// NewCache creates a new Redis cache instance.
// keyPrefix is used to namespace all keys and prevent collisions with other applications.
func NewCache(client *redis.Client, logger *zap.Logger, keyPrefix string) *Cache {
	return &Cache{
		client:    client,
		logger:    logger,
		keyPrefix: keyPrefix,
	}
}

// Get retrieves a value by key. Returns nil if the key doesn't exist.
// The key is automatically prefixed with the configured keyPrefix.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.buildKey(key)

	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		// Key doesn't exist - this is not an error condition
		return nil, nil
	}
	if err != nil {
		c.logger.Error("cache get failed",
			zap.String("key", key),
			zap.Error(err),
		)

		return nil, err
	}

	c.logger.Debug("cache hit",
		zap.String("key", key),
		zap.Int("bytes", len(data)),
	)

	return data, nil
}

// Set stores a value with the given TTL.
// The key is automatically prefixed with the configured keyPrefix.
func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	fullKey := c.buildKey(key)

	err := c.client.Set(ctx, fullKey, value, ttl).Err()
	if err != nil {
		c.logger.Error("cache set failed",
			zap.String("key", key),
			zap.Int("bytes", len(value)),
			zap.Duration("ttl", ttl),
			zap.Error(err),
		)

		return err
	}

	c.logger.Debug("cache set",
		zap.String("key", key),
		zap.Int("bytes", len(value)),
		zap.Duration("ttl", ttl),
	)

	return nil
}

// Delete removes a value by key.
// Returns nil if the key doesn't exist (idempotent operation).
func (c *Cache) Delete(ctx context.Context, key string) error {
	fullKey := c.buildKey(key)

	err := c.client.Del(ctx, fullKey).Err()
	if err != nil {
		c.logger.Error("cache delete failed",
			zap.String("key", key),
			zap.Error(err),
		)

		return err
	}

	c.logger.Debug("cache delete",
		zap.String("key", key),
	)

	return nil
}

// Clear removes all cached values matching the keyPrefix.
// Uses SCAN to find keys, which is safe for production use (non-blocking).
func (c *Cache) Clear(ctx context.Context) error {
	pattern := c.keyPrefix + ":*"

	// Use SCAN to find all keys matching our prefix
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()

	keys := []string{}
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		c.logger.Error("cache clear scan failed",
			zap.String("pattern", pattern),
			zap.Error(err),
		)

		return err
	}

	// Delete all found keys
	if len(keys) > 0 {
		err := c.client.Del(ctx, keys...).Err()
		if err != nil {
			c.logger.Error("cache clear delete failed",
				zap.Int("key_count", len(keys)),
				zap.Error(err),
			)

			return err
		}

		c.logger.Info("cache cleared",
			zap.Int("key_count", len(keys)),
		)
	} else {
		c.logger.Debug("cache clear: no keys found",
			zap.String("pattern", pattern),
		)
	}

	return nil
}

// buildKey creates a fully-qualified key by prefixing with the configured keyPrefix.
func (c *Cache) buildKey(key string) string {
	return c.keyPrefix + ":" + key
}
