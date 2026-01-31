// Package job provides background job schedulers.
package job

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"search-engine-service/internal/app/service"
)

// SyncScheduler runs periodic content synchronization.
type SyncScheduler struct {
	syncService *service.SyncService
	interval    time.Duration
	timeout     time.Duration
	logger      *zap.Logger

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

// NewSyncScheduler creates a new SyncScheduler.
func NewSyncScheduler(syncSvc *service.SyncService, cfg SyncConfig, logger *zap.Logger) *SyncScheduler {
	return &SyncScheduler{
		syncService: syncSvc,
		interval:    cfg.Interval,
		timeout:     cfg.Timeout,
		logger:      logger,
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

// executeSync performs a sync operation with timeout.
func (s *SyncScheduler) executeSync() {
	s.logger.Debug("executing scheduled sync")

	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	results := s.syncService.SyncAll(ctx)

	// Log summary
	totalSynced := 0
	totalErrors := 0
	for _, r := range results {
		if r.Error != nil {
			totalErrors++
			s.logger.Warn("provider sync failed",
				zap.String("provider", r.Provider),
				zap.Error(r.Error),
			)
		} else {
			totalSynced += r.Count
		}
	}

	s.logger.Info("scheduled sync completed",
		zap.Int("total_synced", totalSynced),
		zap.Int("providers_failed", totalErrors),
	)
}
