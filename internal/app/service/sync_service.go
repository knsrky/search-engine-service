package service

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"search-engine-service/internal/domain"
)

// SyncService handles content synchronization from providers.
type SyncService struct {
	repo      domain.ContentRepository
	providers []domain.Provider
	logger    *zap.Logger
}

// NewSyncService creates a new SyncService.
func NewSyncService(repo domain.ContentRepository, providers []domain.Provider, logger *zap.Logger) *SyncService {
	return &SyncService{
		repo:      repo,
		providers: providers,
		logger:    logger,
	}
}

// SyncResult holds the result of a sync operation.
type SyncResult struct {
	Provider string
	Count    int
	Duration time.Duration
	Error    error
}

// SyncAll synchronizes content from all providers concurrently.
// Returns results for each provider. Partial failures are allowed.
func (s *SyncService) SyncAll(ctx context.Context) []SyncResult {
	results := make([]SyncResult, len(s.providers))
	var wg sync.WaitGroup

	s.logger.Info("starting sync from all providers",
		zap.Int("provider_count", len(s.providers)),
	)

	for i, provider := range s.providers {
		wg.Add(1)
		go func(idx int, p domain.Provider) {
			defer wg.Done()
			results[idx] = s.syncProvider(ctx, p)
		}(i, provider)
	}

	wg.Wait()

	// Log summary
	totalSynced := 0
	totalErrors := 0
	for _, r := range results {
		if r.Error != nil {
			totalErrors++
		} else {
			totalSynced += r.Count
		}
	}

	s.logger.Info("sync completed",
		zap.Int("total_synced", totalSynced),
		zap.Int("providers_failed", totalErrors),
	)

	return results
}

// syncProvider fetches and upserts content from a single provider.
func (s *SyncService) syncProvider(ctx context.Context, provider domain.Provider) SyncResult {
	start := time.Now()
	result := SyncResult{
		Provider: provider.Name(),
	}

	s.logger.Debug("syncing provider", zap.String("provider", provider.Name()))

	// Fetch from provider
	contents, err := provider.Fetch(ctx)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(start)
		s.logger.Warn("provider fetch failed",
			zap.String("provider", provider.Name()),
			zap.Error(err),
		)
		return result
	}

	// Bulk upsert to database
	if len(contents) > 0 {
		if err := s.repo.BulkUpsert(ctx, contents); err != nil {
			result.Error = err
			result.Duration = time.Since(start)
			s.logger.Error("bulk upsert failed",
				zap.String("provider", provider.Name()),
				zap.Error(err),
			)
			return result
		}
	}

	result.Count = len(contents)
	result.Duration = time.Since(start)

	s.logger.Info("provider sync completed",
		zap.String("provider", provider.Name()),
		zap.Int("count", result.Count),
		zap.Duration("duration", result.Duration),
	)

	return result
}

// SyncProvider synchronizes content from a specific provider.
func (s *SyncService) SyncProvider(ctx context.Context, providerName string) (*SyncResult, error) {
	for _, p := range s.providers {
		if p.Name() == providerName {
			result := s.syncProvider(ctx, p)
			return &result, result.Error
		}
	}
	return nil, nil // Provider not found
}

// GetProviderNames returns the names of all registered providers.
func (s *SyncService) GetProviderNames() []string {
	names := make([]string, len(s.providers))
	for i, p := range s.providers {
		names[i] = p.Name()
	}
	return names
}
