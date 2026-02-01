// Package job provides background job schedulers.
package job

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"search-engine-service/internal/app/service"
	"search-engine-service/pkg/locker"
)

// SyncScheduler runs periodic content synchronization with distributed locking
// to ensure only one instance executes sync jobs at a time.
type SyncScheduler struct {
	syncService *service.SyncService
	interval    time.Duration
	timeout     time.Duration
	logger      *zap.Logger
	locker      locker.DistributedLocker

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// SyncConfig holds sync scheduler configuration.
type SyncConfig struct {
	Interval  time.Duration
	Timeout   time.Duration
	OnStartup bool
}

// NewSyncScheduler creates a new SyncScheduler with distributed locking support.
//
// Parameters:
//   - syncSvc: Service handling the actual sync operations
//   - cfg: Sync configuration including interval and timeout
//   - logger: Structured logger for operational visibility
//   - locker: Distributed locker for cross-instance coordination
func NewSyncScheduler(
	syncSvc *service.SyncService,
	cfg SyncConfig,
	logger *zap.Logger,
	locker locker.DistributedLocker,
) *SyncScheduler {
	return &SyncScheduler{
		syncService: syncSvc,
		interval:    cfg.Interval,
		timeout:     cfg.Timeout,
		logger:      logger,
		locker:      locker,
	}
}

// Start begins the background sync job.
func (s *SyncScheduler) Start(runOnStartup bool) {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.logger.Info("starting sync scheduler",
		zap.Duration("interval", s.interval),
		zap.Bool("run_on_startup", runOnStartup),
	)

	s.wg.Add(1)
	go s.run(runOnStartup)
}

// Stop gracefully stops the scheduler.
func (s *SyncScheduler) Stop() {
	s.logger.Info("stopping sync scheduler")
	s.cancel()
	s.wg.Wait()
	s.logger.Info("sync scheduler stopped")
}

// run is the main loop of the scheduler.
func (s *SyncScheduler) run(runOnStartup bool) {
	defer s.wg.Done()

	// Run immediately if configured
	if runOnStartup {
		s.executeSync()
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.executeSync()
		}
	}
}

// executeSync performs a sync operation with distributed locking and timeout.
//
// Locking behavior:
//   - Lock TTL = interval duration (cooldown model, not timeout)
//   - Success: Lock held for full interval to prevent duplicate syncs
//   - Failure: Lock released immediately to allow retry by another instance
func (s *SyncScheduler) executeSync() {
	const lockKey = "sync:scheduler:lock"

	// Try to acquire lock with interval-based TTL (cooldown model)
	acquired, err := s.locker.Acquire(s.ctx, lockKey, s.interval)
	if err != nil {
		s.logger.Error("failed to acquire distributed lock", zap.Error(err))

		return
	}
	if !acquired {
		s.logger.Debug("another instance is running sync, skipping execution")

		return
	}

	// Lock acquired - run sync with timeout
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	results := s.syncService.SyncAll(ctx)

	// Analyze results
	totalSynced := 0
	totalErrors := 0
	hasError := false

	for _, r := range results {
		if r.Error != nil {
			totalErrors++
			hasError = true
			s.logger.Warn("provider sync failed",
				zap.String("provider", r.Provider),
				zap.Error(r.Error),
			)
		} else {
			totalSynced += r.Count
		}
	}

	// Handle success vs error scenarios
	if hasError {
		// Release lock immediately on error (allow immediate retry)
		if err := s.locker.Release(s.ctx, lockKey); err != nil {
			s.logger.Error("failed to release lock after sync error", zap.Error(err))
		}
		s.logger.Info("sync completed with errors, lock released for retry",
			zap.Int("total_synced", totalSynced),
			zap.Int("providers_failed", totalErrors),
		)
	} else {
		// Lock will expire naturally after interval (cooldown period)
		s.logger.Info("sync completed successfully, lock held for cooldown",
			zap.Int("total_synced", totalSynced),
			zap.Duration("cooldown", s.interval),
		)
	}
}
