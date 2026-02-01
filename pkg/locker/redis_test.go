package locker

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testLockKey = "test:lock"

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()

	// Create an in-memory Redis instance for testing
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cleanup := func() {
		_ = client.Close()
		mr.Close()
	}

	return client, cleanup
}

func TestRedisLocker_Acquire_Success(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zap.NewNop()
	locker := NewRedisLocker(client, logger)

	ctx := context.Background()
	key := "test:lock"
	ttl := 5 * time.Second

	// First acquisition should succeed
	acquired, err := locker.Acquire(ctx, key, ttl)
	require.NoError(t, err)
	assert.True(t, acquired, "First acquisition should succeed")
}

func TestRedisLocker_Acquire_AlreadyHeld(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zap.NewNop()
	locker1 := NewRedisLocker(client, logger)
	locker2 := NewRedisLocker(client, logger)

	ctx := context.Background()
	key := testLockKey
	ttl := 5 * time.Second

	// First locker acquires the lock
	acquired1, err := locker1.Acquire(ctx, key, ttl)
	require.NoError(t, err)
	assert.True(t, acquired1, "First acquisition should succeed")

	// Second locker should fail to acquire (may return false or error depending on redis state)
	acquired2, _ := locker2.Acquire(ctx, key, ttl)
	assert.False(t, acquired2, "Second acquisition should fail when lock is held")
}

func TestRedisLocker_Release_Success(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zap.NewNop()
	locker := NewRedisLocker(client, logger)

	ctx := context.Background()
	key := testLockKey
	ttl := 5 * time.Second

	// Acquire lock
	acquired, err := locker.Acquire(ctx, key, ttl)
	require.NoError(t, err)
	require.True(t, acquired)

	// Release lock
	err = locker.Release(ctx, key)
	require.NoError(t, err)

	// Should be able to acquire again after release
	acquired2, err := locker.Acquire(ctx, key, ttl)
	require.NoError(t, err)
	assert.True(t, acquired2, "Should be able to acquire after release")
}

func TestRedisLocker_Release_NotOwned(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zap.NewNop()
	locker1 := NewRedisLocker(client, logger)
	locker2 := NewRedisLocker(client, logger)

	ctx := context.Background()
	key := testLockKey
	ttl := 5 * time.Second

	// Locker1 acquires the lock
	acquired, err := locker1.Acquire(ctx, key, ttl)
	require.NoError(t, err)
	require.True(t, acquired)

	// Locker2 tries to release (should not error, but won't release)
	err = locker2.Release(ctx, key)
	require.NoError(t, err)

	// Locker1 should still be able to release (lock still held)
	err = locker1.Release(ctx, key)
	require.NoError(t, err)
}

func TestRedisLocker_ConcurrentAcquisition(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zap.NewNop()
	key := testLockKey
	ttl := 2 * time.Second

	// Simulate 5 instances trying to acquire the lock concurrently
	const numInstances = 5
	results := make(chan bool, numInstances)
	ctx := context.Background()

	for i := 0; i < numInstances; i++ {
		go func() {
			locker := NewRedisLocker(client, logger)
			acquired, _ := locker.Acquire(ctx, key, ttl)
			results <- acquired
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numInstances; i++ {
		if <-results {
			successCount++
		}
	}

	// Exactly one instance should have acquired the lock
	assert.Equal(t, 1, successCount, "Exactly one instance should acquire the lock")
}

func TestRedisLocker_ContextCancellation(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	logger := zap.NewNop()
	locker := NewRedisLocker(client, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	key := testLockKey
	ttl := 5 * time.Second

	// Acquire should fail due to canceled context
	acquired, err := locker.Acquire(ctx, key, ttl)
	assert.Error(t, err)
	assert.False(t, acquired)
}
