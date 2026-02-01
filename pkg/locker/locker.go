// Package locker provides distributed locking capabilities for coordinating
// operations across multiple service instances.
package locker

import (
	"context"
	"time"
)

// DistributedLocker provides distributed lock capabilities across multiple instances.
// Implementations must be safe for concurrent use.
//
// Typical usage:
//
//	acquired, err := locker.Acquire(ctx, "my-lock", 5*time.Minute)
//	if err != nil {
//	    return err
//	}
//	if !acquired {
//	    // Another instance holds the lock
//	    return nil
//	}
//	defer locker.Release(ctx, "my-lock")
//
//	// Perform work while holding the lock
type DistributedLocker interface {
	// Acquire attempts to acquire a distributed lock with the given key.
	// Returns true if the lock was acquired, false if another instance holds it.
	// The lock will automatically expire after ttl if not released.
	//
	// The ttl should be set based on the operation's purpose:
	// - For mutual exclusion: use operation timeout
	// - For cooldown/rate limiting: use the desired cooldown period
	Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Release releases the lock identified by key.
	// Returns an error if the lock doesn't exist or the release fails.
	// Safe to call even if this instance doesn't own the lock (no-op).
	Release(ctx context.Context, key string) error
}
