package locker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisLocker implements DistributedLocker using the Redsync library.
// Redsync implements the Redlock algorithm for distributed mutual exclusion,
// providing production-ready distributed locking with proper failure handling.
type RedisLocker struct {
	rs      *redsync.Redsync
	logger  *zap.Logger
	mutexes map[string]*redsync.Mutex
	mu      sync.Mutex
}

// NewRedisLocker creates a new Redis-based distributed locker using Redsync.
//
// Redsync implements the Redlock algorithm as described in Redis documentation:
// https://redis.io/docs/latest/develop/use/patterns/distributed-locks/
//
// The implementation provides:
// - Atomic lock acquisition and release
// - Automatic expiration to prevent deadlocks
// - Protection against clock drift and network issues
// - Battle-tested reliability (used by Sourcegraph, Google, etc.)
func NewRedisLocker(client *redis.Client, logger *zap.Logger) *RedisLocker {
	pool := goredis.NewPool(client)
	rs := redsync.New(pool)

	return &RedisLocker{
		rs:      rs,
		logger:  logger,
		mutexes: make(map[string]*redsync.Mutex),
	}
}

// Acquire attempts to acquire a distributed lock using the Redlock algorithm.
// Returns true if the lock was acquired, false if another instance holds it.
//
// The lock will automatically expire after ttl if not released, preventing
// deadlocks in case the holder crashes or becomes unreachable.
//
// Implementation details:
// - Uses Redsync's NewMutex with expiry and tries=1 (non-blocking)
// - Returns false (not error) when lock is already held
// - Stores mutex reference for proper release
// - Safe for concurrent use across multiple instances
func (r *RedisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	// Create a mutex with the specified TTL and single try (non-blocking)
	mutex := r.rs.NewMutex(
		key,
		redsync.WithExpiry(ttl),
		redsync.WithTries(1), // Don't retry, return immediately
	)

	// Try to acquire the lock
	err := mutex.LockContext(ctx)
	if err != nil {
		// Check for "lock already taken" errors
		// Redsync can return different error messages for lock contention:
		// 1. redsync.ErrFailed - Standard "lock taken" error
		// 2. Wrapped errors with message "lock already taken, locked nodes: [X]"
		if err == redsync.ErrFailed || strings.Contains(err.Error(), "lock already taken") {
			r.logger.Debug("lock already held by another instance",
				zap.String("key", key),
			)
			return false, nil
		}
		// Real errors (Redis connection issues, context cancellation, etc.)
		return false, fmt.Errorf("acquire lock %s: %w", key, err)
	}

	// Store mutex for later release
	r.mu.Lock()
	r.mutexes[key] = mutex
	r.mu.Unlock()

	r.logger.Debug("lock acquired",
		zap.String("key", key),
		zap.Duration("ttl", ttl),
	)

	return true, nil
}

// Release releases the lock if and only if this instance owns it.
//
// Redsync handles token verification internally, ensuring that:
// - Only the lock holder can release the lock
// - No race conditions between expiration and release
// - Safe to call even if we don't own the lock (no error)
func (r *RedisLocker) Release(ctx context.Context, key string) error {
	r.mu.Lock()
	mutex, exists := r.mutexes[key]
	if exists {
		delete(r.mutexes, key)
	}
	r.mu.Unlock()

	if !exists {
		r.logger.Debug("no mutex found for key, lock not owned by this instance",
			zap.String("key", key),
		)
		return nil
	}

	// Try to release the lock
	ok, err := mutex.UnlockContext(ctx)
	if err != nil {
		return fmt.Errorf("release lock %s: %w", key, err)
	}

	if ok {
		r.logger.Debug("lock released",
			zap.String("key", key),
		)
	} else {
		r.logger.Debug("lock not owned by this instance or already expired",
			zap.String("key", key),
		)
	}

	return nil
}
